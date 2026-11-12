package command_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// handlingPlugins declares a handler of a plugin with a backend, one with items, and one of a plugin
// the host holds suspended.
type handlingPlugins struct{ declaredByTwoPlugins }

func (handlingPlugins) ListHandlers() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "org.example.scans", Command: manifest.Command{ID: "app:file.createText", Title: "Read the scan", When: "target.isFile"}},
		{PluginID: "org.example.scans", Command: manifest.Command{ID: "app:link.downloadMedia", Title: "Download the reading", When: "version.kind == 'reading'"}},
		{PluginID: "org.example.crash", Command: manifest.Command{ID: "app:file.createText", Title: "Crash"}},
	}
}

func (handlingPlugins) GetParts(pluginID string) plugin.Parts {
	if pluginID == "org.example.crash" {
		return plugin.Parts{HasBackend: true, IsSuspended: true}
	}
	return plugin.Parts{HasBackend: true}
}

func TestListCommands_ListsTheHandlersOfThePluginsTheHostRuns(t *testing.T) {
	handler := newHandler(newScenarios(handlingPlugins{}, handlingPlugins{}, nil, nil, nil))

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

// commandReadsCounted is handlingPlugins counting the reads of the commands the plugins declare.
type commandReadsCounted struct {
	handlingPlugins
	reads *int
}

func (p commandReadsCounted) ListCommands() []plugin.CommandEntry {
	*p.reads++
	return p.handlingPlugins.ListCommands()
}

func TestListCommandItems_OfASharedCommandReadsOnlyTheHandlers(t *testing.T) {
	reads := 0
	commands := newScenarios(commandReadsCounted{reads: &reads}, commandReadsCounted{reads: &reads}, nil, nil, nil)
	handler := newHandler(commands.withVersions(&versionItems{}))

	list := listItemsOfOne(t, handler, "org.example.scans", "app:link.downloadMedia", "scans/page.png")

	if len(list.GetItems()) != 2 {
		t.Errorf("got %v, want the two versions", list)
	}
	if reads != 0 {
		t.Errorf("the items of a shared command read the commands of the plugins %d times", reads)
	}
}
