package command_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// keyBindingPlugins binds keys for an enabled plugin with a frontend, an enabled one with a backend,
// and one the host holds suspended.
type keyBindingPlugins struct{ declaredByTwoPlugins }

func (keyBindingPlugins) ListKeybindings() []plugin.KeybindingEntry {
	return []plugin.KeybindingEntry{
		{PluginID: "md.numen.cards", Keybinding: manifest.Keybinding{Key: "mod+shift+d", Command: "deck.create"}},
		{PluginID: "md.numen.down", Keybinding: manifest.Keybinding{Key: "j", Command: "down.run"}},
		{PluginID: "md.numen.tree", Keybinding: manifest.Keybinding{
			Key: "n", Command: "app:note.create", Values: map[string]string{"where": "plex"}, When: "!window.isFieldFocused",
		}},
	}
}

func (keyBindingPlugins) GetParts(pluginID string) plugin.Parts {
	switch pluginID {
	case "md.numen.cards":
		return plugin.Parts{HasBackend: true, HasFrontend: true}
	case "md.numen.down":
		return plugin.Parts{HasBackend: true, IsSuspended: true}
	}
	return plugin.Parts{HasFrontend: true}
}

func TestListCommands_ListsTheKeybindingsOfEnabledPlugins(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(keyBindingPlugins{}, keyBindingPlugins{}, nil, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	keys := res.Msg.GetKeybindings()
	if len(keys) != 2 {
		t.Fatalf("got %v, want the keybindings of the two enabled plugins", keys)
	}
	deck, note := keys[0], keys[1]
	if deck.GetPluginId() != "md.numen.cards" || deck.GetKeybinding().GetKey() != "mod+shift+d" || deck.GetKeybinding().GetCommand() != "deck.create" {
		t.Errorf("the first keybinding came out as %v", deck)
	}
	if note.GetPluginId() != "md.numen.tree" || note.GetKeybinding().GetWhen() != "!window.isFieldFocused" || note.GetKeybinding().GetValues()["where"] != "plex" {
		t.Errorf("the second keybinding came out as %v", note)
	}
}
