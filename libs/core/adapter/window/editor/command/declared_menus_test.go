package command_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// menuPlacingPlugins places commands in menus for an enabled plugin with a frontend and one with a
// frontend and a backend.
type menuPlacingPlugins struct{ declaredByTwoPlugins }

func (menuPlacingPlugins) ListMenus() []plugincommand.MenuEntry {
	order, zero := 10, 0
	return []plugincommand.MenuEntry{
		{PluginID: "md.numen.cards", Menu: manifest.Menu{Location: "titleBar.commands", Command: "deck.create", Group: "new", Order: &order}},
		{PluginID: "md.numen.cards", Menu: manifest.Menu{Location: "searchResult", Command: "app:note.create", Order: &zero}},
		{PluginID: "md.numen.tree", Menu: manifest.Menu{Location: "searchResult", Command: "tree.revealNote", Group: "navigate"}},
	}
}

func (menuPlacingPlugins) GetParts(pluginID string) plugincommand.Parts {
	switch pluginID {
	case "md.numen.cards":
		return plugincommand.Parts{HasBackend: true, HasFrontend: true}
	case "md.numen.tree":
		return plugincommand.Parts{HasFrontend: true}
	}
	return plugincommand.Parts{}
}

func TestListCommands_ListsTheMenuEntriesOfEnabledPluginsInTheOrderOfTheManifests(t *testing.T) {
	handler := command.NewHandler(shownVault{}, plugincommand.NewCommands(menuPlacingPlugins{}, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	menus := res.Msg.GetMenus()
	if len(menus) != 3 {
		t.Fatalf("got %v", menus)
	}
	deck, create, tree := menus[0], menus[1], menus[2]
	if deck.GetPluginId() != "md.numen.cards" || deck.GetLocation() != "titleBar.commands" || deck.GetCommand() != "deck.create" ||
		deck.GetGroup() != "new" || deck.Order == nil || deck.GetOrder() != 10 {
		t.Errorf("the first entry came out as %v", deck)
	}
	if create.GetPluginId() != "md.numen.cards" || create.GetCommand() != "app:note.create" || create.GetGroup() != "" {
		t.Errorf("the second entry came out as %v", create)
	}
	if create.Order == nil || create.GetOrder() != 0 {
		t.Errorf("an order of zero came out as %v, and must differ from no order", create)
	}
	if tree.GetPluginId() != "md.numen.tree" || tree.GetLocation() != "searchResult" || tree.GetGroup() != "navigate" || tree.Order != nil {
		t.Errorf("the third entry came out as %v", tree)
	}
}
