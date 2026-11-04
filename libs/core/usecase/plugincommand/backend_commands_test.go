package plugincommand_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// mixedCommands declares one command of a plugin with a backend and one of a plugin with only a frontend.
type mixedCommands struct{ calls []plugincommand.Call }

func (m *mixedCommands) ListCommands() []plugincommand.Entry {
	return []plugincommand.Entry{
		{PluginID: "md.numen.cards", HasBackend: true, HasFrontend: true, Command: plugincommand.Command{ID: "deck.create", Title: "Create deck"}},
		{PluginID: "md.numen.tree", HasFrontend: true, Command: plugincommand.Command{ID: "tree.open", Title: "Open file tree", Scope: "note"}},
	}
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

func TestTheWindowListsTheCommandsOfEveryPluginWhateverPartRunsThem(t *testing.T) {
	commands := plugincommand.NewCommands(&mixedCommands{}, getTypes(), nil)

	entries := commands.List.ExecuteAll()

	if len(entries) != 2 || entries[1].Command.ID != "tree.open" || entries[1].HasBackend || !entries[1].HasFrontend {
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
