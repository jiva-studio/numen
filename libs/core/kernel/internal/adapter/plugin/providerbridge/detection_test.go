package providerbridge_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	agentsPluginID = "org.example.agents"
	kiteTypeID     = agentsPluginID + "/kite"
	kiteID         = kiteTypeID + "/Kite agent"
)

// detectingSource is the providers of the plugins: each answers connections.detected with the connections set for
// its type, and counts the answers.
type detectingSource struct {
	mu       sync.Mutex
	found    map[string][]modelcall.DetectedConnection
	calls    int
	failOpen error
	failList error
	failCall error
}

func (s *detectingSource) OpenProvider(context.Context, string) (connectionport.Provider, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failOpen != nil {
		return nil, nil, s.failOpen
	}
	return s, func() {}, nil
}

func (s *detectingSource) Call(context.Context, string, modelcall.ProviderCall, []byte, modelcall.Send) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return nil, s.failCall
}

func (s *detectingSource) ListOptions(context.Context, string, modelcall.ProviderCall, []string) ([]modelcall.Option, error) {
	return nil, nil
}

func (s *detectingSource) ListDetected(_ context.Context, call modelcall.ProviderCall) ([]modelcall.DetectedConnection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.failList != nil {
		return nil, s.failList
	}
	return s.found[call.Type], nil
}

func (s *detectingSource) set(typeID string, found ...modelcall.DetectedConnection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.found[typeID] = found
}

func (s *detectingSource) fail(open, list error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failOpen, s.failList = open, list
}

type detectingTypes []connection.Type

func (t detectingTypes) ListTypes() []connection.Type { return t }

func (t detectingTypes) GetType(typeID string) (connection.Type, bool) {
	for _, one := range t {
		if one.GetID() == typeID {
			return one, true
		}
	}
	return connection.Type{}, false
}

func detectingType(pluginID, id string, isAvailable bool) connection.Type {
	return connection.Type{PluginID: pluginID, IsAvailable: isAvailable, Declaration: manifest.ConnectionType{
		ID: id, Title: id, Operations: []string{"acp.task", connection.OperationDetected},
		Fields: []manifest.ConnectionField{
			{Key: "program", Type: "file", Title: "Program", IsRequired: true},
			{Key: "token", Type: manifest.ConnectionFieldSecret, Title: "Token"},
			{Key: "parallelRequests", Type: "number", Title: "Parallel requests", Default: float64(1)},
		},
	}}
}

type detectionFixture struct {
	source    *detectingSource
	detection *providerbridge.Detection
	types     detectingTypes
	clock     *standingClock
	changes   chan struct{}
}

func newDetectionFixture() *detectionFixture {
	f := &detectionFixture{source: &detectingSource{found: map[string][]modelcall.DetectedConnection{
		"kite": {{Name: "Kite agent", Fields: map[string]string{"program": "kite", "parallelRequests": "2", "token": "invented"}}},
		"wren": {{Name: "Wren agent"}},
	}}, changes: make(chan struct{}, 8)}
	types := detectingTypes{
		detectingType(agentsPluginID, "kite", true), detectingType("org.example.gone", "wren", false),
		{PluginID: agentsPluginID, IsAvailable: true, Declaration: manifest.ConnectionType{ID: "heron", Title: "Heron", Operations: []string{"acp.task"}}},
	}
	f.types, f.clock = types, &standingClock{now: time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)}
	f.detection = providerbridge.NewDetection(types, f.source, f.clock)
	f.detection.Follow(func() { f.changes <- struct{}{} }, func(error) {})
	return f
}

func (f *detectionFixture) detect(t *testing.T) {
	t.Helper()
	f.detection.Detect(t.Context(), agentsPluginID)
	f.detection.Detect(t.Context(), "org.example.gone")
}

func TestTheAnswerOfAProviderIsListedForItsAvailableType(t *testing.T) {
	f := newDetectionFixture()

	f.detect(t)

	listed := f.detection.ListDetected()
	if len(listed) != 1 || listed[0].ID != kiteID || listed[0].TypeID != kiteTypeID || listed[0].Name != "Kite agent" {
		t.Fatalf("listed = %+v, want Kite agent alone", listed)
	}
	if fields := listed[0].Fields; len(fields) != 2 || fields["program"] != "kite" || fields["parallelRequests"] != float64(2) {
		t.Errorf("fields = %v, want the program and the parallel requests as a number, and no secret", fields)
	}
}

func TestAProviderThatHasNotAnsweredListsNoConnection(t *testing.T) {
	f := newDetectionFixture()

	if listed := f.detection.ListDetected(); len(listed) != 0 || !f.detection.IsPending(kiteTypeID) {
		t.Errorf("listed = %+v, pending = %v, want none and pending", listed, f.detection.IsPending(kiteTypeID))
	}
	if f.detection.IsPending(agentsPluginID + "/heron") {
		t.Error("a type that does not serve detection is pending")
	}
}

func TestATypeIsPendingWhileItsPluginFailsToStart(t *testing.T) {
	f := newDetectionFixture()
	f.source.fail(errors.New("the plugin did not start"), nil)

	f.detect(t)

	if listed := f.detection.ListDetected(); len(listed) != 0 || !f.detection.IsPending(kiteTypeID) {
		t.Errorf("listed = %+v, want none and the type pending", listed)
	}
}

func TestAFailedAnswerKeepsTheConnectionsOfTheAnswerBefore(t *testing.T) {
	f := newDetectionFixture()
	f.detect(t)
	f.source.fail(nil, &connection.TryNextError{Reason: connection.FailureReasonUnavailable})

	f.detect(t)

	if listed := f.detection.ListDetected(); len(listed) != 1 || listed[0].ID != kiteID {
		t.Errorf("listed = %+v, want Kite agent as before", listed)
	}
}

func TestAnAnswerThatChangesTheConnectionsIsReportedOnce(t *testing.T) {
	f := newDetectionFixture()
	f.detect(t)
	f.detect(t)
	f.source.set("kite")
	f.detect(t)

	if got := len(f.changes); got != 2 {
		t.Errorf("changes = %d, want 2: the first answer and the one that found nothing", got)
	}
	if listed := f.detection.ListDetected(); len(listed) != 0 || f.detection.IsPending(kiteTypeID) {
		t.Errorf("listed = %+v, want none and the type answered", listed)
	}
}

func TestAnAnswerKeepsOneConnectionOfEachValidName(t *testing.T) {
	f := newDetectionFixture()
	f.source.set("kite", modelcall.DetectedConnection{Name: "Kite agent"}, modelcall.DetectedConnection{Name: "kite AGENT"},
		modelcall.DetectedConnection{Name: " Hawk"}, modelcall.DetectedConnection{Name: "Hawk"})

	f.detect(t)

	var names []string
	for _, one := range f.detection.ListDetected() {
		names = append(names, one.Name)
	}
	if !slices.Equal(names, []string{"Kite agent", "Hawk"}) {
		t.Errorf("names = %v", names)
	}
}

func TestRefreshAsksTheProvidersInTheBackground(t *testing.T) {
	f := newDetectionFixture()

	f.detection.Refresh()

	select {
	case <-f.changes:
	case <-time.After(5 * time.Second):
		t.Fatal("the providers were not asked")
	}
	if listed := f.detection.ListDetected(); len(listed) != 1 || listed[0].ID != kiteID {
		t.Errorf("listed = %+v, want Kite agent", listed)
	}
}
