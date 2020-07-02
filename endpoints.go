package relay

import (
	"crypto/sha1" //nolint:gosec // we're not using SHA1 for encryption, just for generating an insecure hash
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	ldeval "gopkg.in/launchdarkly/go-server-sdk-evaluation.v1"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"
	"gopkg.in/launchdarkly/go-server-sdk.v5/interfaces"
	"gopkg.in/launchdarkly/ld-relay.v6/internal/events"
	"gopkg.in/launchdarkly/ld-relay.v6/internal/util"
	"gopkg.in/launchdarkly/ld-relay.v6/logging"
)

// Old stream endpoint that just sends "ping" events: clientstream.ld.com/mping (mobile)
// or clientstream.ld.com/ping/{envId} (JS)
func pingStreamHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		clientCtx := getClientContext(req)
		clientCtx.getLoggers().Debug("Application requested client-side ping stream")
		clientCtx.getHandlers().pingStreamHandler.ServeHTTP(w, req)
	})
}

// Server-side SDK streaming endpoint for both flags and segments: stream.ld.com/all
func allStreamHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		clientCtx := getClientContext(req)
		clientCtx.getLoggers().Debug("Application requested server-side /all stream")
		clientCtx.getHandlers().allStreamHandler.ServeHTTP(w, req)
	})
}

// Old server-side SDK streaming endpoint for just flags: stream.ld.com/flags
func flagsStreamHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		clientCtx := getClientContext(req)
		clientCtx.getLoggers().Debug("Application requested server-side /flags stream")
		clientCtx.getHandlers().flagsStreamHandler.ServeHTTP(w, req)
	})
}

// PHP SDK polling endpoint for all flags: app.ld.com/sdk/flags
func pollAllFlagsHandler(w http.ResponseWriter, req *http.Request) {
	clientCtx := getClientContext(req)
	data, err := clientCtx.getStore().GetAll(interfaces.DataKindFeatures())
	if err != nil {
		clientCtx.getLoggers().Errorf("Error reading feature store: %s", err)
		w.WriteHeader(500)
		return
	}
	respData := itemsCollectionToMap(data)
	// Compute an overall Etag for the data set by hashing flag keys and versions
	hash := sha1.New()                                                         // nolint:gas // just used for insecure hashing
	sort.Slice(data, func(i, j int) bool { return data[i].Key < data[j].Key }) // makes the hash deterministic
	for _, item := range data {
		_, _ = io.WriteString(hash, fmt.Sprintf("%s:%d", item.Key, item.Item.Version))
	}
	etag := hex.EncodeToString(hash.Sum(nil))[:15]
	writeCacheableJSONResponse(w, req, clientCtx, respData, etag)
}

// PHP SDK polling endpoint for a flag: app.ld.com/sdk/flags/{key}
func pollFlagHandler(w http.ResponseWriter, req *http.Request) {
	pollFlagOrSegment(getClientContext(req), interfaces.DataKindFeatures())(w, req)
}

// PHP SDK polling endpoint for a segment: app.ld.com/sdk/segments/{key}
func pollSegmentHandler(w http.ResponseWriter, req *http.Request) {
	pollFlagOrSegment(getClientContext(req), interfaces.DataKindSegments())(w, req)
}

// Event-recorder endpoints:
// events.ld.com/bulk (server-side)
// events.ld.com/diagnostic (server-side diagnostic)
// events.ld.com/mobile, events.ld.com/mobile/events, events.ld.com/mobileevents/bulk (mobile)
// events.ld.com/mobile/events/diagnostic (mobile diagnostic)
// events.ld.com/events/bulk/{envId} (JS)
// events.ld.com/events/diagnostic/{envId} (JS)
func bulkEventHandler(endpoint events.Endpoint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		clientCtx := getClientContext(req)
		dispatcher := clientCtx.getHandlers().eventDispatcher
		if dispatcher == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write(util.ErrorJsonMsg("Event proxy is not enabled for this environment"))
			return
		}
		handler := dispatcher.GetHandler(endpoint)
		if handler == nil {
			// Note, if this ever happens, it is a programming error since we are only supposed to
			// be using a fixed set of Endpoint values that the dispatcher knows about.
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write(util.ErrorJsonMsg("Internal error in event proxy"))
			logging.GlobalLoggers.Errorf("Tried to proxy events for unsupported endpoint '%s'", endpoint)
			return
		}
		handler(w, req)
	})
}

// Client-side evaluation endpoint, new schema with metadata:
// app.ld.com/sdk/evalx/{envId}/users/{user} (GET)
// app.ld.com/sdk/evalx/{envId}/user (REPORT)
// app.ld/com/sdk/evalx/users/{user} (GET - with SDK key auth)
// app.ld/com/sdk/evalx/user (REPORT - with SDK key auth)
func evaluateAllFeatureFlags(sdkKind sdkKind) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		evaluateAllShared(w, req, false, sdkKind)
	}
}

// Client-side evaluation endpoint, old schema with only values:
// app.ld.com/sdk/eval/{envId}/users/{user} (GET)
// app.ld.com/sdk/eval/{envId}/user (REPORT)
// app.ld/com/sdk/eval/users/{user} (GET - with SDK key auth)
// app.ld/com/sdk/eval/user (REPORT - with SDK key auth)
func evaluateAllFeatureFlagsValueOnly(sdkKind sdkKind) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		evaluateAllShared(w, req, true, sdkKind)
	}
}

func evaluateAllShared(w http.ResponseWriter, req *http.Request, valueOnly bool, sdkKind sdkKind) {
	var user lduser.User
	var userDecodeErr error
	if req.Method == "REPORT" {
		if req.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte("Content-Type must be application/json."))
			return
		}

		body, _ := ioutil.ReadAll(req.Body)
		userDecodeErr = json.Unmarshal(body, &user)
	} else {
		base64User := mux.Vars(req)["user"]
		user, userDecodeErr = UserV2FromBase64(base64User)
	}
	if userDecodeErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(util.ErrorJsonMsg(userDecodeErr.Error()))
		return
	}

	withReasons := req.URL.Query().Get("withReasons") == "true"

	clientCtx := getClientContext(req)
	client := clientCtx.getClient()
	store := clientCtx.getStore()
	loggers := clientCtx.getLoggers()

	w.Header().Set("Content-Type", "application/json")

	if !client.Initialized() {
		if store.IsInitialized() {
			loggers.Warn("Called before client initialization; using last known values from feature store")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			loggers.Warn("Called before client initialization. Feature store not available")
			w.Write(util.ErrorJsonMsg("Service not initialized"))
			return
		}
	}

	if user.GetKey() == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(util.ErrorJsonMsg("User must have a 'key' attribute"))
		return
	}

	loggers.Debugf("Application requested client-side flags (%s) for user: %s", sdkKind, user.GetKey())

	items, err := store.GetAll(interfaces.DataKindFeatures())
	if err != nil {
		loggers.Warnf("Unable to fetch flags from feature store. Returning nil map. Error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(util.ErrorJsonMsgf("Error fetching flags from feature store: %s", err))
		return
	}

	evaluator := ldeval.NewEvaluator(basicDataProvider{store})

	response := make(map[string]interface{}, len(items))
	for _, item := range items {
		if flag, ok := item.Item.Item.(*ldmodel.FeatureFlag); ok {
			if sdkKind == jsClientSdk && !flag.ClientSide {
				continue
			}
			detail := evaluator.Evaluate(flag, user, nil)
			var result interface{}
			if valueOnly {
				result = detail.JSONValue
			} else {
				isExperiment := flag.IsExperimentationEnabled(detail.Reason)
				value := evalXResult{
					Value:                detail.Value,
					Version:              flag.Version,
					TrackEvents:          flag.TrackEvents || isExperiment,
					DebugEventsUntilDate: flag.DebugEventsUntilDate,
					TrackReason:          isExperiment,
				}
				if detail.VariationIndex >= 0 {
					value.Variation = &detail.VariationIndex
				}
				if withReasons || isExperiment {
					value.Reason = &detail.Reason
				}
				result = value
			}
			response[flag.Key] = result
		}
	}

	result, _ := json.Marshal(response)

	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

type basicDataProvider struct {
	store interfaces.DataStore
}

func (p basicDataProvider) GetFeatureFlag(key string) *ldmodel.FeatureFlag {
	data, err := p.store.Get(interfaces.DataKindFeatures(), key)
	if err == nil && data.Item != nil {
		if f, ok := data.Item.(*ldmodel.FeatureFlag); ok {
			return f
		}
	}
	return nil
}

func (p basicDataProvider) GetSegment(key string) *ldmodel.Segment {
	data, err := p.store.Get(interfaces.DataKindSegments(), key)
	if err == nil && data.Item != nil {
		if s, ok := data.Item.(*ldmodel.Segment); ok {
			return s
		}
	}
	return nil
}

func pollFlagOrSegment(clientContext clientContext, kind interfaces.StoreDataKind) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		key := mux.Vars(req)["key"]
		item, err := clientContext.getStore().Get(kind, key)
		if err != nil {
			clientContext.getLoggers().Errorf("Error reading feature store: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if item.Item == nil {
			w.WriteHeader(http.StatusNotFound)
		} else {
			writeCacheableJSONResponse(w, req, clientContext, item.Item, strconv.Itoa(item.Version))
		}
	}
}

func writeCacheableJSONResponse(w http.ResponseWriter, req *http.Request, clientContext clientContext,
	entity interface{}, etagValue string) {
	etag := fmt.Sprintf("relay-%s", etagValue) // just to make it extra clear that these are relay-specific etags
	if cachedEtag := req.Header.Get("If-None-Match"); cachedEtag != "" {
		if cachedEtag == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	bytes, err := json.Marshal(entity)
	if err != nil {
		clientContext.getLoggers().Errorf("Error marshaling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Etag", etag)
		ttl := clientContext.getTtl()
		if ttl > 0 {
			w.Header().Set("Vary", "Authorization")
			expiresAt := time.Now().UTC().Add(ttl)
			w.Header().Set("Expires", expiresAt.Format(http.TimeFormat))
			// We're setting "Expires:" instead of "Cache-Control:max-age=" so that if someone puts an
			// HTTP cache in front of ld-relay, multiple clients hitting the cache at different times
			// will all see the same expiration time.
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes)
	}
}

// getUserAgent returns the X-LaunchDarkly-User-Agent if available, falling back to the normal "User-Agent" header
func getUserAgent(req *http.Request) string {
	if agent := req.Header.Get(ldUserAgentHeader); agent != "" {
		return agent
	}
	return req.Header.Get(userAgentHeader)
}

var hexdigit = regexp.MustCompile(`[a-fA-F\d]`)

func obscureKey(key string) string {
	if len(key) > 8 {
		return key[0:4] + hexdigit.ReplaceAllString(key[4:len(key)-5], "*") + key[len(key)-5:]
	}
	return key
}

func itemsCollectionToMap(coll []interfaces.StoreKeyedItemDescriptor) map[string]interface{} {
	ret := make(map[string]interface{}, len(coll))
	for _, item := range coll {
		ret[item.Key] = item.Item.Item
	}
	return ret
}
