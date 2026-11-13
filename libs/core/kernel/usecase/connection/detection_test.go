package connection_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/connectiontest"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

const (
	agentsPluginID = "org.example.agents"
	kiteTypeID     = agentsPluginID + "/kite"
	kiteID         = kiteTypeID + "/Kite agent"
	wrenID         = "org.example.gone/wren/Wren agent"
)

// detectingSource is the providers of the plugins: each answers connections.detected with the connections set for
// its type, and counts the calls.
type detectingSource struct {
	mu       sync.Mutex
	found    map[string][]modelcall.DetectedConnection
	calls    int
	failOpen error
	failList error
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
	return nil, nil
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

func (s *detectingSource) countCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func detectingType(pluginID, id string, isAvailable bool) connectiondomain.Type {
	return connectiondomain.Type{PluginID: pluginID, IsAvailable: isAvailable, Declaration: manifest.ConnectionType{
		ID: id, Title: id, Operations: []string{taskOp, connectiondomain.OperationDetected},
		RowFields: []manifest.ConnectionField{{Key: "model", Type: "string", Title: "Model"}},
		Fields: []manifest.ConnectionField{
			{Key: "program", Type: "file", Title: "Program", IsRequired: true},
			{Key: "parallelRequests", Type: "number", Title: "Parallel requests", Default: float64(1), Min: new(1.0), Step: new(1.0)},
		},
	}}
}

type detectionFixture struct {
	*fixture
	source    *detectingSource
	detection *connection.Detection
	changes   chan struct{}
}

func newDetectionFixture(t *testing.T) *detectionFixture {
	t.Helper()
	f := &detectionFixture{fixture: newFixture(t), source: &detectingSource{found: map[string][]modelcall.DetectedConnection{
		"kite": {{Name: "Kite agent", Fields: map[string]string{"program": "kite", "parallelRequests": "2"}}},
		"wren": {{Name: "Wren agent"}},
	}}, changes: make(chan struct{}, 8)}
	f.types.types = append(f.types.types, detectingType(agentsPluginID, "kite", true), detectingType("org.example.gone", "wren", false))
	f.detection = connection.NewDetection(f.types, f.source)
	f.detection.Follow(func() { f.changes <- struct{}{} }, func(error) {})
	next := 0
	f.connection = connectiontest.NewWithDetected(f.repository, f.types, f.keys, f.uses, f.detection, func() (string, error) {
		next++
		return fmt.Sprintf("c%08x", next), nil
	})
	return f
}

func (f *detectionFixture) detect(t *testing.T) {
	t.Helper()
	f.detection.Detect(t.Context(), agentsPluginID)
	f.detection.Detect(t.Context(), "org.example.gone")
}

func TestListShowsTheDetectedConnectionsAfterTheStoredOnes(t *testing.T) {
	f := newDetectionFixture(t)
	stored := f.createOpenRouter(t, "Work")
	f.detect(t)

	entries := f.listEntries(t)

	if len(entries) != 2 || entries[0].ID != stored.ID || entries[0].IsDetected {
		t.Fatalf("entries = %+v, want Work and then Kite agent", entries)
	}
	kite := entries[1]
	if kite.ID != kiteID || kite.TypeID != kiteTypeID || kite.Name != "Kite agent" || !kite.IsDetected || !kite.IsComplete ||
		kite.Fields["program"] != "kite" {
		t.Errorf("the detected connection = %+v", kite)
	}
	if len(f.listStoredConnections(t)) != 1 {
		t.Errorf("stored = %+v, want the detected connection left out of the file", f.listStoredConnections(t))
	}
}

func TestAProviderThatHasNotAnsweredListsNoConnection(t *testing.T) {
	f := newDetectionFixture(t)

	if listed := f.detection.ListDetected(); len(listed) != 0 || !f.detection.IsPending(kiteTypeID) {
		t.Errorf("listed = %+v, pending = %v, want none and pending", listed, f.detection.IsPending(kiteTypeID))
	}
}

func TestAPluginThatCannotStartLeavesItsTypePending(t *testing.T) {
	f := newDetectionFixture(t)
	f.source.failOpen = errors.New("the plugin did not start")

	f.detect(t)

	if listed := f.detection.ListDetected(); len(listed) != 0 || !f.detection.IsPending(kiteTypeID) {
		t.Errorf("listed = %+v, want none and the type pending", listed)
	}
}

func TestAFailedAnswerKeepsTheConnectionsOfTheAnswerBefore(t *testing.T) {
	f := newDetectionFixture(t)
	f.detect(t)
	f.source.failList = &connectiondomain.TryNextError{Reason: connectiondomain.FailureReasonUnavailable}

	f.detect(t)

	if listed := f.detection.ListDetected(); len(listed) != 1 || listed[0].ID != kiteID {
		t.Errorf("listed = %+v, want Kite agent as before", listed)
	}
}

func TestAnAnswerThatChangesTheConnectionsIsReportedOnce(t *testing.T) {
	f := newDetectionFixture(t)
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
	f := newDetectionFixture(t)
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

func TestListAsksTheProvidersInTheBackground(t *testing.T) {
	f := newDetectionFixture(t)

	f.listEntries(t)

	select {
	case <-f.changes:
	case <-time.After(5 * time.Second):
		t.Fatal("the list did not ask the providers")
	}
	if entries := f.listEntries(t); len(entries) != 1 || entries[0].ID != kiteID {
		t.Errorf("entries = %+v, want Kite agent", entries)
	}
}

func TestADetectedConnectionCannotBeChangedOrDeleted(t *testing.T) {
	f := newDetectionFixture(t)
	f.detect(t)

	_, updateErr := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: kiteID, Name: "Hawk"})
	_, deleteErr := f.connection.Delete.Execute(t.Context(), kiteID)

	if !errors.Is(updateErr, connectiondomain.ErrNotFound) || !errors.Is(deleteErr, connectiondomain.ErrNotFound) {
		t.Errorf("update = %v, delete = %v, want ErrNotFound for both", updateErr, deleteErr)
	}
}

func TestANewConnectionCannotTakeTheNameOfADetectedOne(t *testing.T) {
	f := newDetectionFixture(t)
	f.detect(t)

	_, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: openRouterID, Name: "kite agent", Secrets: map[string]string{"apiKey": "sk-or-1"},
	})

	if !errors.Is(err, connection.ErrNameTaken) {
		t.Errorf("err = %v, want ErrNameTaken", err)
	}
}

func TestAStoredConnectionOfADetectingTypeIsAdded(t *testing.T) {
	f := newDetectionFixture(t)
	f.detect(t)

	created, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: kiteTypeID, Name: "Kite large", Fields: map[string]any{"program": "kite"},
	})

	if err != nil || created.IsDetected {
		t.Errorf("created = %+v, %v", created, err)
	}
}

func TestADetectedConnectionResolvesWithItsFieldsOnceItsProviderAnswered(t *testing.T) {
	f := newDetectionFixture(t)

	resolved, err := f.connection.Resolve.Execute(kiteID)
	if err != nil {
		t.Fatal(err)
	}
	secrets, err := f.connection.ReadSecrets.Execute(t.Context(), resolved)
	if err != nil {
		t.Fatal(err)
	}

	if resolved.PluginID != agentsPluginID || resolved.ParallelRequests != 2 || resolved.Fields["program"] != "kite" || len(secrets) != 0 {
		t.Errorf("resolved = %+v, secrets = %v", resolved, secrets)
	}
	if _, err := f.connection.Resolve.Execute(wrenID); !errors.Is(err, connectiondomain.ErrNotFound) {
		t.Errorf("the detected connection of a disabled plugin: err = %v, want ErrNotFound", err)
	}
}

func TestListRowsDiscoversTheDetectedConnectionsAfterTheStoredOnes(t *testing.T) {
	repository := &memoryRepository{connections: []connectiondomain.Connection{
		{ID: "c00000002", TypeID: "org.example.agents/heron", Name: "Heron"},
	}}
	types := fixedTypes{types: []connectiondomain.Type{
		agentType(agentsPluginID, "heron", true, taskOp),
		detectingType(agentsPluginID, "kite", true),
		detectingType("org.example.gone", "wren", false),
	}}
	source := &detectingSource{found: map[string][]modelcall.DetectedConnection{"kite": {{Name: "Kite agent"}}, "wren": {{Name: "Wren agent"}}}}
	deps := connection.NewDeps(repository, types, &memoryKeys{values: map[string]string{}}, &memoryUses{}).
		WithDetected(connection.NewDetection(types, source))

	rows, err := connection.NewListRows(deps, fixedRowSettings{setting: connectiondomain.RowSetting{IsDiscovering: true}}).Execute(chatPluginID, taskOp)
	if err != nil {
		t.Fatal(err)
	}

	want := []connectiondomain.Row{
		{ConnectionID: "c00000002", Options: map[string]string{}, IsDiscovered: true},
		{ConnectionID: kiteID, Options: map[string]string{}, IsDiscovered: true},
	}
	if !reflect.DeepEqual(rows, want) || source.countCalls() != 1 {
		t.Errorf("rows = %v, calls = %d, want Heron and then Kite agent after one call", rows, source.countCalls())
	}
}
