package connection_test

import (
	"reflect"
	"testing"

	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

const (
	chatPluginID = "md.numen.views.agent.chat"
	taskOp       = "acp.task"
)

// fixedRowSettings is the setting of rows of the chat as the person left it.
type fixedRowSettings struct{ setting connectiondomain.RowSetting }

func (s fixedRowSettings) ReadRowSetting(pluginID, operation string) (connectiondomain.RowSetting, error) {
	if pluginID != chatPluginID || operation != taskOp {
		return connectiondomain.RowSetting{}, nil
	}
	return s.setting, nil
}

func agentType(pluginID, id string, isAvailable bool, operations ...string) connectiondomain.Type {
	return connectiondomain.Type{PluginID: pluginID, IsAvailable: isAvailable, Declaration: manifest.ConnectionType{ID: id, Title: id, Operations: operations}}
}

// listRows lists the rows of the chat over a list of connections of agents, models and a type no plugin
// provides any longer.
func listRows(t *testing.T, setting connectiondomain.RowSetting) []connectiondomain.Row {
	t.Helper()
	repository := &memoryRepository{connections: []connectiondomain.Connection{
		{ID: "c00000001", TypeID: "org.example.models/chat", Name: "Lantern Models"},
		{ID: "c00000002", TypeID: "org.example.agents/heron", Name: "Heron"},
		{ID: "c00000003", TypeID: "org.example.retired/owl", Name: "Owl"},
		{ID: "c00000004", TypeID: "org.example.agents/finch", Name: "Finch"},
		{ID: "c00000005", TypeID: "org.example.gone/wren", Name: "Wren"},
	}}
	types := fixedTypes{types: []connectiondomain.Type{
		agentType("org.example.models", "chat", true, "llm.chat"),
		agentType("org.example.agents", "heron", true, taskOp),
		agentType("org.example.agents", "finch", true, "llm.chat", taskOp),
		agentType("org.example.retired", "owl", false, taskOp),
	}}
	deps := connection.NewDeps(repository, types, &memoryKeys{values: map[string]string{}}, &memoryUses{})
	rows, err := connection.NewListRows(deps, fixedRowSettings{setting: setting}).Execute(chatPluginID, taskOp)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestTheRowsThePersonChoseAreUsedWhileTheSettingDiscovers(t *testing.T) {
	chosen := []connectiondomain.Row{{ConnectionID: "c00000004", Options: map[string]string{"model": "finch-large"}}}

	rows := listRows(t, connectiondomain.RowSetting{Rows: chosen, IsDiscovering: true})

	if !reflect.DeepEqual(rows, chosen) {
		t.Errorf("rows = %v, want the chosen %v", rows, chosen)
	}
}

func TestASettingThatDiscoversUsesEveryAvailableConnectionThatServesTheOperationInTheirOrder(t *testing.T) {
	rows := listRows(t, connectiondomain.RowSetting{IsDiscovering: true})

	want := []connectiondomain.Row{
		{ConnectionID: "c00000002", Options: map[string]string{}},
		{ConnectionID: "c00000004", Options: map[string]string{}},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want Heron and Finch with no options", rows)
	}
}

func TestASettingThatDoesNotDiscoverAndHoldsNoRowUsesNoConnection(t *testing.T) {
	rows := listRows(t, connectiondomain.RowSetting{})

	if len(rows) != 0 {
		t.Errorf("rows = %v, want none", rows)
	}
}
