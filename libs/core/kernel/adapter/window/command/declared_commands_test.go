package command_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// declaredByTwoPlugins declares a command of a plugin with only a frontend and one of a plugin with a backend.
type declaredByTwoPlugins struct{}

func (declaredByTwoPlugins) ListKeybindings() []plugin.KeybindingEntry { return nil }

func (declaredByTwoPlugins) ListSlots() []plugin.SlotEntry { return nil }

func (declaredByTwoPlugins) ListHandlers() []plugin.CommandEntry { return nil }

func (declaredByTwoPlugins) ListCommands() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "md.numen.tree", Command: manifest.Command{
			ID: "tree.revealNote", Title: "Show note in file tree", Icon: "folder-open", When: "target.isNote",
		}},
		{PluginID: "md.numen.cards", Command: manifest.Command{ID: "deck.create", Title: "Create deck"}},
	}
}

func (declaredByTwoPlugins) GetParts(pluginID string) plugin.Parts {
	if pluginID == "md.numen.cards" {
		return plugin.Parts{HasBackend: true, HasFrontend: true}
	}
	return plugin.Parts{HasFrontend: true}
}

func (declaredByTwoPlugins) RunCommand(context.Context, plugin.CommandCall) (plugin.CommandResult, error) {
	return plugin.CommandResult{Message: "ran"}, nil
}

func TestListCommands_CarriesTheDeclarationOfEachCommand(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(declaredByTwoPlugins{}, declaredByTwoPlugins{}, nil, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	items := res.Msg.GetCommands()
	if len(items) != 2 {
		t.Fatalf("got %v", items)
	}
	tree := items[0]
	if tree.GetWhen() != "target.isNote" {
		t.Errorf("the frontend command came out as %v", tree)
	}
}

func TestRunPluginCommand_AnswersUnknownForACommandOfAPluginWithoutABackend(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(declaredByTwoPlugins{}, declaredByTwoPlugins{}, nil, nil, nil))

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

func (suspendedPlugin) GetParts(string) plugin.Parts {
	return plugin.Parts{HasBackend: true, HasFrontend: true, IsSuspended: true}
}

func TestListCommands_ListsNoCommandOfASuspendedPlugin(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(suspendedPlugin{}, suspendedPlugin{}, nil, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if items := res.Msg.GetCommands(); len(items) != 0 {
		t.Errorf("got %v", items)
	}
}
