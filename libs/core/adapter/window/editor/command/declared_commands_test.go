package command_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// declaredByTwoPlugins declares a command of a plugin with only a frontend and one of a plugin with a backend.
type declaredByTwoPlugins struct{}

func (declaredByTwoPlugins) ListCommands() []plugincommand.Entry {
	return []plugincommand.Entry{
		{PluginID: "md.numen.tree", Command: plugincommand.Command{
			ID: "tree.revealNote", Title: "Show note in file tree", Icon: "folder-open", Scope: "note", Shortcut: "mod+shift+e",
		}},
		{PluginID: "md.numen.cards", Command: plugincommand.Command{ID: "deck.create", Title: "Create deck"}},
	}
}

func (declaredByTwoPlugins) GetParts(pluginID string) plugincommand.Parts {
	if pluginID == "md.numen.cards" {
		return plugincommand.Parts{HasBackend: true, HasFrontend: true}
	}
	return plugincommand.Parts{HasFrontend: true}
}

func (declaredByTwoPlugins) RunCommand(context.Context, plugincommand.Call) (plugincommand.Result, error) {
	return plugincommand.Result{Message: "ran"}, nil
}

func TestListCommands_CarriesTheDeclarationOfEachCommand(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(declaredByTwoPlugins{}, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	items := res.Msg.GetCommands()
	if len(items) != 2 {
		t.Fatalf("got %v", items)
	}
	tree := items[0]
	if tree.GetScope() != "note" || tree.GetShortcut() != "mod+shift+e" {
		t.Errorf("the frontend command came out as %v", tree)
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

func (suspendedPlugin) GetParts(string) plugincommand.Parts {
	return plugincommand.Parts{HasBackend: true, HasFrontend: true, IsSuspended: true}
}

func TestListCommands_ListsNoCommandOfASuspendedPlugin(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(suspendedPlugin{}, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if items := res.Msg.GetCommands(); len(items) != 0 {
		t.Errorf("got %v", items)
	}
}
