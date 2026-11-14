package connection_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"testing"

	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection/connectiontest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

const (
	agentsPluginID = "org.example.agents"
	kiteTypeID     = agentsPluginID + "/kite"
	kiteID         = kiteTypeID + "/Kite agent"
	wrenID         = "org.example.gone/wren/Wren agent"
)

// answeredDetection stands for the providers of the agents plugin: the type kite is pending until the plugin is
// asked, and then lists Kite agent.
type answeredDetection struct {
	mu        sync.Mutex
	isAsked   bool
	refreshes int
	asked     []string
}

func (d *answeredDetection) ListDetected() []connectiondomain.Connection {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.isAsked {
		return nil
	}
	return []connectiondomain.Connection{{
		ID: kiteID, TypeID: kiteTypeID, Name: "Kite agent", Fields: map[string]any{"program": "kite", "parallelRequests": float64(2)},
	}}
}

func (d *answeredDetection) IsPending(typeID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return typeID == kiteTypeID && !d.isAsked
}

func (d *answeredDetection) Detect(_ context.Context, pluginID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.asked = append(d.asked, pluginID)
	d.isAsked = d.isAsked || pluginID == agentsPluginID
}

func (d *answeredDetection) Refresh() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.refreshes++
}

func (d *answeredDetection) listAskedPlugins() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.asked)
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
	detection *answeredDetection
}

func newDetectionFixture(t *testing.T) *detectionFixture {
	t.Helper()
	f := &detectionFixture{fixture: newFixture(t), detection: &answeredDetection{}}
	f.types.types = append(f.types.types, detectingType(agentsPluginID, "kite", true), detectingType("org.example.gone", "wren", false))
	next := 0
	f.connection = connectiontest.NewWithDetection(f.repository, f.types, f.keys, f.uses, f.detection, func() (string, error) {
		next++
		return fmt.Sprintf("c%08x", next), nil
	})
	return f
}

func TestListShowsTheDetectedConnectionsAfterTheStoredOnes(t *testing.T) {
	f := newDetectionFixture(t)
	stored := f.createOpenRouter(t, "Work")
	f.detection.Detect(t.Context(), agentsPluginID)

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

func TestListAsksTheProvidersToDetectAgain(t *testing.T) {
	f := newDetectionFixture(t)

	f.listEntries(t)
	f.listEntries(t)

	if f.detection.refreshes != 2 {
		t.Errorf("refreshes = %d, want one for each list", f.detection.refreshes)
	}
}

func TestADetectedConnectionCannotBeChangedOrDeleted(t *testing.T) {
	f := newDetectionFixture(t)
	f.detection.Detect(t.Context(), agentsPluginID)

	_, updateErr := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: kiteID, Name: "Hawk"})
	_, deleteErr := f.connection.Delete.Execute(t.Context(), kiteID)

	if !errors.Is(updateErr, connectiondomain.ErrNotFound) || !errors.Is(deleteErr, connectiondomain.ErrNotFound) {
		t.Errorf("update = %v, delete = %v, want ErrNotFound for both", updateErr, deleteErr)
	}
}

func TestANewConnectionCannotTakeTheNameOfADetectedOne(t *testing.T) {
	f := newDetectionFixture(t)
	f.detection.Detect(t.Context(), agentsPluginID)

	_, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: openRouterID, Name: "kite agent", Secrets: map[string]string{"apiKey": "sk-or-1"},
	})

	if !errors.Is(err, connection.ErrNameTaken) {
		t.Errorf("err = %v, want ErrNameTaken", err)
	}
}

func TestAStoredConnectionOfADetectingTypeIsAdded(t *testing.T) {
	f := newDetectionFixture(t)
	f.detection.Detect(t.Context(), agentsPluginID)

	created, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: kiteTypeID, Name: "Kite large", Fields: map[string]any{"program": "kite"},
	})

	if err != nil || created.IsDetected {
		t.Errorf("created = %+v, %v", created, err)
	}
}

func TestADetectedConnectionOfAPendingTypeResolvesOnceItsProviderIsAsked(t *testing.T) {
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
	if asked := f.detection.listAskedPlugins(); !slices.Equal(asked, []string{agentsPluginID}) {
		t.Errorf("asked = %v, want the agents plugin once", asked)
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
	detection := &answeredDetection{}
	deps := connection.NewDeps(repository, types, &memoryKeys{values: map[string]string{}}, &memoryUses{}).AddDetection(detection)

	rows, err := connection.NewListRows(deps, fixedRowSettings{setting: connectiondomain.RowSetting{IsDiscovering: true}}).Execute(chatPluginID, taskOp)
	if err != nil {
		t.Fatal(err)
	}

	want := []connectiondomain.Row{
		{ConnectionID: "c00000002", Options: map[string]string{}, IsDiscovered: true},
		{ConnectionID: kiteID, Options: map[string]string{}, IsDiscovered: true},
	}
	if !reflect.DeepEqual(rows, want) || !slices.Equal(detection.listAskedPlugins(), []string{agentsPluginID}) {
		t.Errorf("rows = %v, asked = %v, want Heron and then Kite agent after one question", rows, detection.listAskedPlugins())
	}
}
