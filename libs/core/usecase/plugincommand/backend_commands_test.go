package plugincommand_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// mixedCommands declares commands of plugins with a backend and a frontend, with a frontend only,
// with neither, and of one the host holds suspended.
type mixedCommands struct{ calls []plugincommand.Call }

func (m *mixedCommands) ListMenus() []plugincommand.MenuEntry { return nil }

func (m *mixedCommands) ListCommands() []plugincommand.Entry {
	return []plugincommand.Entry{
		{PluginID: "md.numen.cards", Command: plugincommand.Command{ID: "deck.create", Title: "Create deck"}},
		{PluginID: "md.numen.tree", Command: plugincommand.Command{ID: "tree.open", Title: "Open file tree", Scope: "note"}},
		{PluginID: "md.numen.empty", Command: plugincommand.Command{ID: "empty.run", Title: "Run"}},
		{PluginID: "md.numen.down", Command: plugincommand.Command{ID: "down.run", Title: "Run"}},
	}
}

func (m *mixedCommands) GetParts(pluginID string) plugincommand.Parts {
	return map[string]plugincommand.Parts{
		"md.numen.cards": {HasBackend: true, HasFrontend: true},
		"md.numen.tree":  {HasFrontend: true},
		"md.numen.down":  {HasBackend: true, IsSuspended: true},
	}[pluginID]
}

func (m *mixedCommands) RunCommand(_ context.Context, call plugincommand.Call) (plugincommand.Result, error) {
	m.calls = append(m.calls, call)
	return plugincommand.Result{Message: "ran " + call.Command}, nil
}

func TestTheCommandLineListsOnlyCommandsOfPluginsThatHaveABackend(t *testing.T) {
	commands := plugincommand.NewCommands(&mixedCommands{}, getTypes(), nil)

	entries := commands.List.Execute()

	if len(entries) != 1 || entries[0].Command.ID != "deck.create" {
		t.Fatalf("the command line lists %+v", entries)
	}
}

func TestTheWindowListsTheCommandsOfEveryRunningPluginWhateverPartRunsThem(t *testing.T) {
	commands := plugincommand.NewCommands(&mixedCommands{}, getTypes(), nil)

	entries := commands.List.ExecuteAll()

	if len(entries) != 2 || entries[0].Command.ID != "deck.create" || entries[1].Command.ID != "tree.open" {
		t.Fatalf("the window lists %+v", entries)
	}
}

func TestACommandOfAPluginWithoutABackendIsNotRunInABackend(t *testing.T) {
	declared := &mixedCommands{}

	_, err := plugincommand.NewCommands(declared, getTypes(), nil).Run.Execute(t.Context(), getVault(), plugincommand.Call{
		PluginID: "md.numen.tree", Command: "tree.open",
	})

	if !errors.Is(err, plugincommand.ErrUnknownCommand) || len(declared.calls) != 0 {
		t.Fatalf("the run answered %v and reached the plugin with %+v", err, declared.calls)
	}
}

func TestACommandOfAPluginWithABackendIsRunInItsBackend(t *testing.T) {
	declared := &mixedCommands{}

	result, err := plugincommand.NewCommands(declared, getTypes(), nil).Run.Execute(t.Context(), getVault(), plugincommand.Call{
		PluginID: "md.numen.cards", Command: "deck.create",
	})

	if err != nil || result.Message != "ran deck.create" {
		t.Fatalf("the run answered %+v, %v", result, err)
	}
}
