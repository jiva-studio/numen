package command_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// handlingPlugins declares a handler of a plugin with a backend, one with items, and one of a plugin
// the host holds suspended.
type handlingPlugins struct{ declaredByTwoPlugins }

func (handlingPlugins) ListHandlers() []plugincommand.Entry {
	return []plugincommand.Entry{
		{PluginID: "org.example.scans", Command: plugincommand.Command{ID: "app:file.createText", Title: "Read the scan", When: "target.isFile"}},
		{PluginID: "org.example.scans", Command: plugincommand.Command{ID: "app:version.delete", Title: "Delete the reading", When: "version.kind == 'reading'"}},
		{PluginID: "org.example.crash", Command: plugincommand.Command{ID: "app:file.createText", Title: "Crash"}},
	}
}

func (handlingPlugins) GetParts(pluginID string) plugincommand.Parts {
	if pluginID == "org.example.crash" {
		return plugincommand.Parts{HasBackend: true, IsSuspended: true}
	}
	return plugincommand.Parts{HasBackend: true}
}

func TestListCommands_ListsTheHandlersOfThePluginsTheHostRuns(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(handlingPlugins{}, nil, nil), nil)

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	got := res.Msg.GetHandlers()
	if len(got) != 2 {
		t.Fatalf("got %v, want the two handlers of org.example.scans", got)
	}
	if got[0].GetPluginId() != "org.example.scans" || got[0].GetHandler().GetCommand() != "app:file.createText" ||
		got[0].GetHandler().GetTitle() != "Read the scan" || got[0].GetHandler().GetWhen() != "target.isFile" || got[0].GetHasItems() {
		t.Errorf("the first handler came out as %v", got[0])
	}
	if !got[1].GetHasItems() {
		t.Errorf("the handler over versions came out as %v", got[1])
	}
}
