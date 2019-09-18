package relay

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	ld "gopkg.in/launchdarkly/go-server-sdk.v4"
	"gopkg.in/launchdarkly/go-server-sdk.v4/ldlog"
	"gopkg.in/launchdarkly/ld-relay.v5/internal/events"
	"gopkg.in/launchdarkly/ld-relay.v5/internal/util"
	"gopkg.in/launchdarkly/ld-relay.v5/logging"
)

// Old stream endpoint that just sends "ping" events: clientstream.ld.com/mping (mobile)
// or clientstream.ld.com/ping/{envId} (JS)
func pingStreamHandler(w http.ResponseWriter, req *http.Request) {
	clientCtx := getClientContext(req)
	clientCtx.getLoggers().Debug("Application requested client-side ping stream")
	clientCtx.getHandlers().pingStreamHandler.ServeHTTP(w, req)
}

// Server-side SDK streaming endpoint for both flags and segments: stream.ld.com/all
func allStreamHandler(w http.ResponseWriter, req *http.Request) {
	clientCtx := getClientContext(req)
	clientCtx.getLoggers().Debug("Application requested server-side /all stream")
	clientCtx.getHandlers().allStreamHandler.ServeHTTP(w, req)
}

// Old server-side SDK streaming endpoint for just flags: stream.ld.com/flags
func flagsStreamHandler(w http.ResponseWriter, req *http.Request) {
	clientCtx := getClientContext(req)
	clientCtx.getLoggers().Debug("Application requested server-side /flags stream")
	clientCtx.getHandlers().flagsStreamHandler.ServeHTTP(w, req)
}

// PHP SDK polling endpoint for all flags: app.ld.com/sdk/flags
func pollAllFlagsHandler(w http.ResponseWriter, req *http.Request) {
	clientCtx := getClientContext(req)
	loggers := clientCtx.getLoggers()
	data, err := clientCtx.getStore().All(ld.Features)
	if err != nil {
		loggers.Errorf("Error reading feature store: %s", err)
		w.WriteHeader(500)
		return
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		loggers.Errorf("Error marshaling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, _ = w.Write(bytes)
}

// PHP SDK polling endpoint for a flag: app.ld.com/sdk/flags/{key}
func pollFlagHandler(w http.ResponseWriter, req *http.Request) {
	clientCtx := getClientContext(req)
	pollFlagOrSegment(clientCtx.getStore(), clientCtx.getLoggers(), ld.Features)(w, req)
}

// PHP SDK polling endpoint for a segment: app.ld.com/sdk/segments/{key}
func pollSegmentHandler(w http.ResponseWriter, req *http.Request) {
	clientCtx := getClientContext(req)
	pollFlagOrSegment(clientCtx.getStore(), clientCtx.getLoggers(), ld.Segments)(w, req)
}

// Event-recorder endpoints:
// events.ld.com/bulk (server-side)
// events.ld.com/mobile, events.ld.com/mobile/events, events.ld.com/mobileevents/bulk (mobile)
// events.ld.com/events/bulk/{envId} (JS)
func bulkEventHandler(endpoint events.Endpoint) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
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
			logging.Error.Printf("Tried to proxy events for unsupported endpoint '%s'", endpoint)
			return
		}
		handler(w, req)
	}
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
	var user *ld.User
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
		if store.Initialized() {
			loggers.Warn("Called before client initialization; using last known values from feature store")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			loggers.Warn("Called before client initialization. Feature store not available")
			w.Write(util.ErrorJsonMsg("Service not initialized"))
			return
		}
	}

	if user.Key == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(util.ErrorJsonMsg("User must have a 'key' attribute"))
		return
	}

	loggers.Debugf("Application requested client-side flags (%s) for user: %s", sdkKind, *user.Key)

	items, err := store.All(ld.Features)
	if err != nil {
		loggers.Warnf("Unable to fetch flags from feature store. Returning nil map. Error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(util.ErrorJsonMsgf("Error fetching flags from feature store: %s", err))
		return
	}

	response := make(map[string]interface{}, len(items))
	for _, item := range items {
		if flag, ok := item.(*ld.FeatureFlag); ok {
			if sdkKind == jsClientSdk && !flag.ClientSide {
				continue
			}
			detail, _ := flag.EvaluateDetail(*user, store, false)
			var result interface{}
			if valueOnly {
				result = detail.Value
			} else {
				value := evalXResult{
					Value:                detail.Value,
					Variation:            detail.VariationIndex,
					Version:              flag.Version,
					TrackEvents:          flag.TrackEvents,
					DebugEventsUntilDate: flag.DebugEventsUntilDate,
				}
				if withReasons {
					value.Reason = &ld.EvaluationReasonContainer{Reason: detail.Reason}
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

func pollFlagOrSegment(featureStore ld.FeatureStore, loggers *ldlog.Loggers, kind ld.VersionedDataKind) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		key := mux.Vars(req)["key"]
		item, err := featureStore.Get(kind, key)
		if err != nil {
			loggers.Errorf("Error reading feature store: %s", err)
			w.WriteHeader(500)
			return
		}
		if item == nil {
			w.WriteHeader(404)
		} else {
			bytes, err := json.Marshal(item)
			if err != nil {
				loggers.Errorf("Error marshaling JSON: %s", err)
				w.WriteHeader(500)
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				_, _ = w.Write(bytes)
			}
		}
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
