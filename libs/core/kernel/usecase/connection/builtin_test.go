package connection_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/connectiontest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

const (
	kiteID = "org.example.agents/kite"
	wrenID = "org.example.gone/wren"
)

func builtInType(pluginID, id, title string, isAvailable bool) connectiondomain.Type {
	return connectiondomain.Type{PluginID: pluginID, IsAvailable: isAvailable, Declaration: manifest.ConnectionType{
		ID: id, Title: title, Operations: []string{taskOp}, IsBuiltIn: true,
		RowFields: []manifest.ConnectionField{{Key: "model", Type: "string", Title: "Model"}},
		Fields: []manifest.ConnectionField{
			{Key: "parallelRequests", Type: "number", Title: "Parallel requests", Default: float64(2), Min: new(1.0), Step: new(1.0)},
		},
	}}
}

func newBuiltInFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	f.types.types = append(f.types.types,
		builtInType("org.example.agents", "kite", "Kite", true),
		builtInType("org.example.gone", "wren", "Wren", false),
	)
	next := 0
	f.connection = connectiontest.New(f.repository, f.types, f.keys, f.uses, func() (string, error) {
		next++
		return fmt.Sprintf("c%08x", next), nil
	})
	return f
}

func TestListShowsTheConnectionOfAnEnabledBuiltInTypeAfterTheStoredOnes(t *testing.T) {
	f := newBuiltInFixture(t)
	stored := f.createOpenRouter(t, "Work")

	entries := f.listEntries(t)

	if len(entries) != 2 || entries[0].ID != stored.ID || entries[0].IsBuiltIn {
		t.Fatalf("entries = %+v, want Work and then Kite", entries)
	}
	kite := entries[1]
	if kite.ID != kiteID || kite.TypeID != kiteID || kite.Name != "Kite" || !kite.IsBuiltIn || !kite.IsComplete || !kite.IsTypeAvailable {
		t.Errorf("the built-in connection = %+v", kite)
	}
	if len(f.listStoredConnections(t)) != 1 {
		t.Errorf("stored = %+v, want the built-in connection left out of the file", f.listStoredConnections(t))
	}
}

func TestABuiltInConnectionCannotBeChangedOrDeleted(t *testing.T) {
	f := newBuiltInFixture(t)

	_, updateErr := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: kiteID, Name: "Hawk"})
	_, deleteErr := f.connection.Delete.Execute(t.Context(), kiteID)

	if !errors.Is(updateErr, connectiondomain.ErrNotFound) || !errors.Is(deleteErr, connectiondomain.ErrNotFound) {
		t.Errorf("update = %v, delete = %v, want ErrNotFound for both", updateErr, deleteErr)
	}
	if entries := f.listEntries(t); len(entries) != 1 || entries[0].Name != "Kite" {
		t.Errorf("entries = %+v, want Kite as it was", entries)
	}
}

func TestANewConnectionDoesNotTakeTheNameOfABuiltInOne(t *testing.T) {
	f := newBuiltInFixture(t)

	_, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: openRouterID, Name: "kite", Secrets: map[string]string{"apiKey": "sk-or-1"},
	})

	if !errors.Is(err, connection.ErrNameTaken) {
		t.Errorf("err = %v, want ErrNameTaken", err)
	}
}

func TestABuiltInConnectionResolvesWithTheDefaultsOfItsType(t *testing.T) {
	f := newBuiltInFixture(t)

	resolved, err := f.connection.Resolve.Execute(kiteID)
	if err != nil {
		t.Fatal(err)
	}
	secrets, err := f.connection.ReadSecrets.Execute(t.Context(), resolved)
	if err != nil {
		t.Fatal(err)
	}

	if resolved.PluginID != "org.example.agents" || resolved.ParallelRequests != 2 || len(resolved.Fields) != 0 || len(secrets) != 0 {
		t.Errorf("resolved = %+v, secrets = %v", resolved, secrets)
	}
	if _, err := f.connection.Resolve.Execute(wrenID); !errors.Is(err, connectiondomain.ErrNotFound) {
		t.Errorf("the built-in connection of a disabled plugin: err = %v, want ErrNotFound", err)
	}
}

func TestListRowsDiscoversTheBuiltInConnectionsAfterTheStoredOnes(t *testing.T) {
	repository := &memoryRepository{connections: []connectiondomain.Connection{
		{ID: "c00000002", TypeID: "org.example.agents/heron", Name: "Heron"},
	}}
	types := fixedTypes{types: []connectiondomain.Type{
		agentType("org.example.agents", "heron", true, taskOp),
		builtInType("org.example.agents", "kite", "Kite", true),
		builtInType("org.example.gone", "wren", "Wren", false),
	}}
	deps := connection.NewDeps(repository, types, &memoryKeys{values: map[string]string{}}, &memoryUses{})

	rows, err := connection.NewListRows(deps, fixedRowSettings{setting: connectiondomain.RowSetting{IsDiscovering: true}}).Execute(chatPluginID, taskOp)
	if err != nil {
		t.Fatal(err)
	}

	want := []connectiondomain.Row{
		{ConnectionID: "c00000002", Options: map[string]string{}},
		{ConnectionID: kiteID, Options: map[string]string{}},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want Heron and then Kite", rows)
	}
}
