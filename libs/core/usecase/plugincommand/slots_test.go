package plugincommand_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// menuPlugins places commands in menus for plugins with a frontend, with a backend, with neither, and
// for one the host holds suspended.
type menuPlugins struct{}

func (menuPlugins) ListCommands() []plugincommand.Entry { return nil }

func (menuPlugins) ListSlots() []plugincommand.SlotEntry {
	order := 20
	return []plugincommand.SlotEntry{
		{PluginID: "org.example.atlas", Entry: manifest.SlotEntry{Slot: "titleBar.commands", Command: "atlas.open", Group: "navigate", Order: &order}},
		{PluginID: "org.example.atlas", Entry: manifest.SlotEntry{Slot: "palette.searchResult", Command: "atlas.open"}},
		{PluginID: "org.example.bare", Entry: manifest.SlotEntry{Slot: "titleBar.commands", Command: "bare.run"}},
		{PluginID: "org.example.crash", Entry: manifest.SlotEntry{Slot: "titleBar.commands", Command: "crash.run"}},
		{PluginID: "org.example.worker", Entry: manifest.SlotEntry{Slot: "titleBar.commands", Command: "worker.run"}},
	}
}

func (menuPlugins) GetParts(pluginID string) plugincommand.Parts {
	return map[string]plugincommand.Parts{
		"org.example.atlas":  {HasFrontend: true},
		"org.example.crash":  {HasBackend: true, HasFrontend: true, IsSuspended: true},
		"org.example.worker": {HasBackend: true},
	}[pluginID]
}

func (menuPlugins) RunCommand(context.Context, plugincommand.Call) (plugincommand.Result, error) {
	return plugincommand.Result{}, nil
}

func TestTheWindowListsTheMenuEntriesOfPluginsThatRunAndNoOthers(t *testing.T) {
	commands := plugincommand.NewCommands(menuPlugins{}, getTypes(), nil)

	entries := commands.List.ExecuteSlots()

	var got []string
	for _, one := range entries {
		got = append(got, one.PluginID+" "+one.Entry.Slot+" "+one.Entry.Command)
	}
	want := []string{
		"org.example.atlas titleBar.commands atlas.open",
		"org.example.atlas palette.searchResult atlas.open",
		"org.example.worker titleBar.commands worker.run",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the window lists %v, want %v", got, want)
	}
}

func TestAMenuEntryKeepsItsGroupAndOrder(t *testing.T) {
	commands := plugincommand.NewCommands(menuPlugins{}, getTypes(), nil)

	entries := commands.List.ExecuteSlots()

	if len(entries) == 0 {
		t.Fatal("the window lists no menu entries")
	}
	first := entries[0].Entry
	if first.Group != "navigate" || first.Order == nil || *first.Order != 20 {
		t.Errorf("the entry came out as %+v", first)
	}
}

func TestTheWindowListsNoMenuEntriesWhereThereIsNoPluginHost(t *testing.T) {
	commands := plugincommand.NewCommands(nil, getTypes(), nil)

	if entries := commands.List.ExecuteSlots(); len(entries) != 0 {
		t.Errorf("the window lists %+v", entries)
	}
}
