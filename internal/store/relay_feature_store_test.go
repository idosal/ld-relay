package store

import (
	"testing"

	"github.com/stretchr/testify/assert"

	es "github.com/launchdarkly/eventsource"
	"github.com/launchdarkly/ld-relay/v6/internal/sharedtest"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldlog"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk.v5/interfaces/ldstoretypes"
	"gopkg.in/launchdarkly/go-server-sdk.v5/ldcomponents/ldstoreimpl"
)

type testPublisher struct {
	events   []es.Event
	comments []string
}

func (p *testPublisher) Publish(channels []string, event es.Event) {
	p.events = append(p.events, event)
}

func (p *testPublisher) PublishComment(channels []string, text string) {
	p.comments = append(p.comments, text)
}

func (p *testPublisher) Register(channel string, repo es.Repository) {}

func TestRelayFeatureStore(t *testing.T) {
	loggers := ldlog.Loggers{}
	loggers.SetMinLevel(ldlog.None)

	t.Run("init", func(t *testing.T) {
		baseStore := sharedtest.NewInMemoryStore()
		allPublisher := &testPublisher{}
		flagsPublisher := &testPublisher{}
		pingPublisher := &testPublisher{}
		store := NewSSERelayFeatureStore("api-key", allPublisher, flagsPublisher, pingPublisher, baseStore, loggers, 1)

		store.Init(nil)
		emptyDataMap := map[string]interface{}{}
		emptyAllMap := map[string]map[string]interface{}{"flags": emptyDataMap, "segments": emptyDataMap}
		assert.EqualValues(t, []es.Event{allPutEvent{D: emptyAllMap}}, allPublisher.events)
		assert.EqualValues(t, []es.Event{flagsPutEvent(emptyDataMap)}, flagsPublisher.events)
		assert.EqualValues(t, []es.Event{pingEvent{}}, pingPublisher.events)
	})

	t.Run("delete flag", func(t *testing.T) {
		baseStore := sharedtest.NewInMemoryStore()
		baseStore.Init(nil)
		allPublisher := &testPublisher{}
		flagsPublisher := &testPublisher{}
		pingPublisher := &testPublisher{}
		store := NewSSERelayFeatureStore("api-key", allPublisher, flagsPublisher, pingPublisher, baseStore, loggers, 1)

		_, _ = store.Upsert(ldstoreimpl.Features(), "my-flag",
			ldstoretypes.ItemDescriptor{Version: 1, Item: nil})
		assert.EqualValues(t, []es.Event{deleteEvent{Path: "/flags/my-flag", Version: 1}}, allPublisher.events)
		assert.EqualValues(t, []es.Event{deleteEvent{Path: "/my-flag", Version: 1}}, flagsPublisher.events)
		assert.EqualValues(t, []es.Event{pingEvent{}}, pingPublisher.events)
	})

	t.Run("create flag", func(t *testing.T) {
		baseStore := sharedtest.NewInMemoryStore()
		baseStore.Init(nil)
		allPublisher := &testPublisher{}
		flagsPublisher := &testPublisher{}
		pingPublisher := &testPublisher{}
		store := NewSSERelayFeatureStore("api-key", allPublisher, flagsPublisher, pingPublisher, baseStore, loggers, 1)

		newFlag := ldbuilders.NewFlagBuilder("my-new-flag").Version(1).Build()
		_, _ = sharedtest.UpsertFlag(store, newFlag)
		assert.EqualValues(t, []es.Event{upsertEvent{Path: "/flags/my-new-flag", D: &newFlag}}, allPublisher.events)
		assert.EqualValues(t, []es.Event{upsertEvent{Path: "/my-new-flag", D: &newFlag}}, flagsPublisher.events)
		assert.EqualValues(t, []es.Event{pingEvent{}}, pingPublisher.events)
	})

	t.Run("update flag", func(t *testing.T) {
		baseStore := sharedtest.NewInMemoryStore()
		baseStore.Init(nil)
		originalFlag := ldbuilders.NewFlagBuilder("my-flag").Version(1).Build()
		_, _ = sharedtest.UpsertFlag(baseStore, originalFlag)

		allPublisher := &testPublisher{}
		flagsPublisher := &testPublisher{}
		pingPublisher := &testPublisher{}
		store := NewSSERelayFeatureStore("api-key", allPublisher, flagsPublisher, pingPublisher, baseStore, loggers, 1)

		updatedFlag := ldbuilders.NewFlagBuilder("my-flag").Version(2).Build()
		_, _ = sharedtest.UpsertFlag(store, updatedFlag)
		assert.EqualValues(t, []es.Event{upsertEvent{Path: "/flags/my-flag", D: &updatedFlag}}, allPublisher.events)
		assert.EqualValues(t, []es.Event{upsertEvent{Path: "/my-flag", D: &updatedFlag}}, flagsPublisher.events)
		assert.EqualValues(t, []es.Event{pingEvent{}}, pingPublisher.events)
	})

	t.Run("updating flag with older version just sends newer version", func(t *testing.T) {
		baseStore := sharedtest.NewInMemoryStore()
		baseStore.Init(nil)
		originalFlag := ldbuilders.NewFlagBuilder("my-flag").Version(2).Build()
		_, _ = sharedtest.UpsertFlag(baseStore, originalFlag)

		allPublisher := &testPublisher{}
		flagsPublisher := &testPublisher{}
		pingPublisher := &testPublisher{}
		store := NewSSERelayFeatureStore("api-key", allPublisher, flagsPublisher, pingPublisher, baseStore, loggers, 1)

		staleFlag := ldbuilders.NewFlagBuilder("my-flag").Version(1).Build()
		_, _ = sharedtest.UpsertFlag(store, staleFlag)
		assert.EqualValues(t, []es.Event{
			upsertEvent{Path: "/flags/my-flag", D: &originalFlag},
		}, allPublisher.events)
		assert.EqualValues(t, []es.Event{
			upsertEvent{Path: "/my-flag", D: &originalFlag},
		}, flagsPublisher.events)
		assert.EqualValues(t, []es.Event{
			pingEvent{},
		}, pingPublisher.events)
	})

	t.Run("updating deleted flag with older version does nothing", func(t *testing.T) {
		baseStore := sharedtest.NewInMemoryStore()
		baseStore.Init(nil)
		_, _ = baseStore.Upsert(ldstoreimpl.Features(), "my-flag",
			ldstoretypes.ItemDescriptor{Version: 2, Item: nil})

		allPublisher := &testPublisher{}
		flagsPublisher := &testPublisher{}
		pingPublisher := &testPublisher{}
		store := NewSSERelayFeatureStore("api-key", allPublisher, flagsPublisher, pingPublisher, baseStore, loggers, 1)

		staleFlag := ldbuilders.NewFlagBuilder("my-flag").Version(1).Build()
		_, _ = sharedtest.UpsertFlag(store, staleFlag)
		assert.EqualValues(t, []es.Event(nil), allPublisher.events)
		assert.EqualValues(t, []es.Event(nil), flagsPublisher.events)
		assert.EqualValues(t, []es.Event(nil), pingPublisher.events)
	})
}
