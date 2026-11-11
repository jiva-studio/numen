package plugincommand_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// mixedCommands declares commands of plugins with a backend and a frontend, with a frontend only,
// with neither, and of one the host holds suspended.
type mixedCommands struct{ calls []plugin.CommandCall }

func (m *mixedCommands) ListKeybindings() []plugin.KeybindingEntry { return nil }

func (m *mixedCommands) ListSlots() []plugin.SlotEntry { return nil }

func (m *mixedCommands) ListHandlers() []plugin.CommandEntry { return nil }

func (m *mixedCommands) ListCommands() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "md.numen.cards", Command: manifest.Command{ID: "deck.create", Title: "Create deck"}},
		{PluginID: "md.numen.tree", Command: manifest.Command{ID: "tree.open", Title: "Open file tree", When: "target.isNote"}},
		{PluginID: "md.numen.empty", Command: manifest.Command{ID: "empty.run", Title: "Run"}},
		{PluginID: "md.numen.down", Command: manifest.Command{ID: "down.run", Title: "Run"}},
	}
}

func (m *mixedCommands) GetParts(pluginID string) plugin.Parts {
	return map[string]plugin.Parts{
		"md.numen.cards": {HasBackend: true, HasFrontend: true},
		"md.numen.tree":  {HasFrontend: true},
		"md.numen.down":  {HasBackend: true, IsSuspended: true},
	}[pluginID]
}

func (m *mixedCommands) RunCommand(_ context.Context, call plugin.CommandCall) (plugin.CommandResult, error) {
	m.calls = append(m.calls, call)
	return plugin.CommandResult{Message: "ran " + call.Command}, nil
}

func TestTheCommandLineListsOnlyCommandsOfPluginsThatHaveABackend(t *testing.T) {
	commands := newScenarios(&mixedCommands{}, &mixedCommands{}, getTypes(), nil, nil)

	entries := commands.List.Execute()

	if len(entries) != 1 || entries[0].Command.ID != "deck.create" {
		t.Fatalf("the command line lists %+v", entries)
	}
}

func TestTheWindowListsTheCommandsOfEveryRunningPluginWhateverPartRunsThem(t *testing.T) {
	commands := newScenarios(&mixedCommands{}, &mixedCommands{}, getTypes(), nil, nil)

	entries := commands.List.ExecuteAll()

	if len(entries) != 2 || entries[0].Command.ID != "deck.create" || entries[1].Command.ID != "tree.open" {
		t.Fatalf("the window lists %+v", entries)
	}
}

func TestACommandOfAPluginWithoutABackendIsNotRunInABackend(t *testing.T) {
	declared := &mixedCommands{}

	_, err := newScenarios(declared, declared, getTypes(), nil, nil).Run.Execute(t.Context(), getVault(), plugin.CommandCall{
		PluginID: "md.numen.tree", Command: "tree.open",
	})

	if !errors.Is(err, plugincommand.ErrUnknownCommand) || len(declared.calls) != 0 {
		t.Fatalf("the run answered %v and reached the plugin with %+v", err, declared.calls)
	}
}

func TestACommandOfAPluginWithABackendIsRunInItsBackend(t *testing.T) {
	declared := &mixedCommands{}

	result, err := newScenarios(declared, declared, getTypes(), nil, nil).Run.Execute(t.Context(), getVault(), plugin.CommandCall{
		PluginID: "md.numen.cards", Command: "deck.create",
	})

	if err != nil || result.Message != "ran deck.create" {
		t.Fatalf("the run answered %+v, %v", result, err)
	}
}
