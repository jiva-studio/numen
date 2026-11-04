package command_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// declaredByTwoPlugins declares a command of a plugin with only a frontend and one of a plugin with a backend.
type declaredByTwoPlugins struct{}

func (declaredByTwoPlugins) ListCommands() []plugincommand.Entry {
	return []plugincommand.Entry{
		{PluginID: "md.numen.tree", HasFrontend: true, Command: plugincommand.Command{
			ID: "tree.revealNote", Title: "Show note in file tree", Icon: "folder-open", Scope: "note", Shortcut: "mod+shift+e", IsNoteDestination: true,
		}},
		{PluginID: "md.numen.cards", HasBackend: true, HasFrontend: true, Command: plugincommand.Command{ID: "deck.create", Title: "Create deck"}},
	}
}

func (declaredByTwoPlugins) RunCommand(context.Context, plugincommand.Call) (plugincommand.Result, error) {
	return plugincommand.Result{Message: "ran"}, nil
}

func TestListCommands_CarriesTheDeclarationAndThePartsOfEachPlugin(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(declaredByTwoPlugins{}, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	items := res.Msg.GetCommands()
	if len(items) != 2 {
		t.Fatalf("got %v", items)
	}
	tree, cards := items[0], items[1]
	if tree.GetScope() != "note" || tree.GetShortcut() != "mod+shift+e" || !tree.GetIsNoteDestination() || tree.GetHasBackend() || !tree.GetHasFrontend() {
		t.Errorf("the frontend command came out as %v", tree)
	}
	if !cards.GetHasBackend() || !cards.GetHasFrontend() {
		t.Errorf("the command of a plugin with both parts came out as %v", cards)
	}
}

func TestRunPluginCommand_AnswersUnknownForACommandOfAPluginWithoutABackend(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(declaredByTwoPlugins{}, nil, nil))

	res, err := handler.RunPluginCommand(t.Context(), connect.NewRequest(&v1.RunPluginCommandRequest{PluginId: "md.numen.tree", Command: "tree.revealNote"}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND {
		t.Errorf("the run answered %v", res.Msg)
	}
}

// suspendedPlugin declares a command of a plugin the host holds suspended after a failed start.
type suspendedPlugin struct{ declaredByTwoPlugins }

func (suspendedPlugin) ListCommands() []plugincommand.Entry {
	return []plugincommand.Entry{{
		PluginID: "md.numen.cards", HasBackend: true, HasFrontend: true, IsSuspended: true,
		Command: plugincommand.Command{ID: "deck.create", Title: "Create deck"},
	}}
}

func TestListCommands_MarksTheCommandsOfASuspendedPlugin(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(suspendedPlugin{}, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	items := res.Msg.GetCommands()
	if len(items) != 1 || !items[0].GetHasBackend() || !items[0].GetHasFrontend() || !items[0].GetIsSuspended() {
		t.Errorf("got %v", items)
	}
}

// conditionalFrontendCommands declares commands of a plugin with only a frontend that carry conditions or note types.
type conditionalFrontendCommands struct{ declaredByTwoPlugins }

func (conditionalFrontendCommands) ListCommands() []plugincommand.Entry {
	return []plugincommand.Entry{
		{PluginID: "md.numen.tree", HasFrontend: true, Command: plugincommand.Command{
			ID: "tree.versioned", Title: "Versioned", Shortcut: "mod+shift+v",
			When: []pluginport.PluginContextCondition{{Key: "version.live", Is: []string{"true"}}},
		}},
		{PluginID: "md.numen.tree", HasFrontend: true, Command: plugincommand.Command{ID: "tree.typed", Title: "Typed", Shortcut: "mod+shift+t", NoteTypes: []string{"deck"}}},
	}
}

func TestListCommands_ListsTheFrontendCommandsTheMenusListWithTheirShortcuts(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(conditionalFrontendCommands{}, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	items := res.Msg.GetCommands()
	if len(items) != 2 {
		t.Fatalf("got %v", items)
	}
	for _, one := range items {
		if !one.GetIsListedByMenu() || one.GetShortcut() == "" {
			t.Errorf("the command %s lost its shortcut or is not marked: %v", one.GetId(), one)
		}
	}
}
