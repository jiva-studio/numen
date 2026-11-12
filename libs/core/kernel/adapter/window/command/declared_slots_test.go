package command_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// menuPlacingPlugins places commands in menus for an enabled plugin with a frontend, an enabled one
// with a backend, and one the host holds suspended.
type menuPlacingPlugins struct{ declaredByTwoPlugins }

func (menuPlacingPlugins) ListSlots() []plugin.SlotEntry {
	order, zero := 10, 0
	return []plugin.SlotEntry{
		{PluginID: "md.numen.cards", Entry: manifest.SlotEntry{Slot: "titleBar.commands", Command: "deck.create", Group: "new", Order: &order}},
		{PluginID: "md.numen.cards", Entry: manifest.SlotEntry{Slot: "palette.searchResult", Command: "app:note.create", Order: &zero, Values: map[string]string{"where": "plex"}}},
		{PluginID: "md.numen.down", Entry: manifest.SlotEntry{Slot: "titleBar.commands", Command: "down.run"}},
		{PluginID: "md.numen.tree", Entry: manifest.SlotEntry{Slot: "palette.searchResult", Command: "tree.revealNote", Group: "navigate"}},
	}
}

func (menuPlacingPlugins) GetParts(pluginID string) plugin.Parts {
	switch pluginID {
	case "md.numen.cards":
		return plugin.Parts{HasBackend: true, HasFrontend: true}
	case "md.numen.down":
		return plugin.Parts{HasBackend: true, IsSuspended: true}
	case "md.numen.tree":
		return plugin.Parts{HasFrontend: true}
	}
	return plugin.Parts{}
}

func TestListCommands_ListsTheMenuEntriesOfEnabledPluginsInTheOrderOfTheManifests(t *testing.T) {
	handler := newHandler(newScenarios(menuPlacingPlugins{}, menuPlacingPlugins{}, nil, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	menus := res.Msg.GetSlots()
	if len(menus) != 3 {
		t.Fatalf("got %v", menus)
	}
	deck, create, tree := menus[0], menus[1], menus[2]
	if deck.GetPluginId() != "md.numen.cards" || deck.GetEntry().GetSlot() != "titleBar.commands" || deck.GetEntry().GetCommand() != "deck.create" ||
		deck.GetEntry().GetGroup() != "new" || deck.GetEntry().Order == nil || deck.GetEntry().GetOrder() != 10 {
		t.Errorf("the first entry came out as %v", deck)
	}
	if create.GetPluginId() != "md.numen.cards" || create.GetEntry().GetCommand() != "app:note.create" || create.GetEntry().GetGroup() != "" {
		t.Errorf("the second entry came out as %v", create)
	}
	if got := create.GetEntry().GetValues(); len(got) != 1 || got["where"] != "plex" {
		t.Errorf("the values of the second entry came out as %v", got)
	}
	if len(deck.GetEntry().GetValues()) != 0 {
		t.Errorf("an entry without values came out with %v", deck.GetEntry().GetValues())
	}
	if create.GetEntry().Order == nil || create.GetEntry().GetOrder() != 0 {
		t.Errorf("an order of zero came out as %v, and must differ from no order", create)
	}
	if tree.GetPluginId() != "md.numen.tree" || tree.GetEntry().GetSlot() != "palette.searchResult" || tree.GetEntry().GetGroup() != "navigate" || tree.GetEntry().Order != nil {
		t.Errorf("the third entry came out as %v", tree)
	}
}

func TestListCommands_ListsNoMenuEntryOfASuspendedPlugin(t *testing.T) {
	handler := newHandler(newScenarios(menuPlacingPlugins{}, menuPlacingPlugins{}, nil, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	for _, one := range res.Msg.GetSlots() {
		if one.GetPluginId() == "md.numen.down" {
			t.Errorf("the suspended plugin has the entry %v", one)
		}
	}
	if len(res.Msg.GetSlots()) == 0 {
		t.Error("the enabled plugins have no entries")
	}
}

// viewAndSubmenuPlugins places a view and a submenu for an enabled plugin with a frontend.
type viewAndSubmenuPlugins struct{ menuPlacingPlugins }

func (viewAndSubmenuPlugins) ListSlots() []plugin.SlotEntry {
	return []plugin.SlotEntry{
		{PluginID: "md.numen.tree", Entry: manifest.SlotEntry{Slot: "titleBar.status", View: "balance"}},
		{PluginID: "md.numen.tree", Entry: manifest.SlotEntry{Slot: "fileTree.context", Submenu: "ledger.new", Title: "New ledger", Icon: "plus", Group: "0-new", When: "isFolder && !isMultiple"}},
	}
}

func TestListCommands_CarriesAViewAndASubmenuWithTheirTitleIconAndCondition(t *testing.T) {
	handler := newHandler(newScenarios(viewAndSubmenuPlugins{}, viewAndSubmenuPlugins{}, nil, nil, nil))

	res, err := handler.ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	slots := res.Msg.GetSlots()
	if len(slots) != 2 {
		t.Fatalf("got %v", slots)
	}
	view, submenu := slots[0].GetEntry(), slots[1].GetEntry()
	if view.GetSlot() != "titleBar.status" || view.GetView() != "balance" || view.GetCommand() != "" {
		t.Errorf("the view came out as %v", view)
	}
	if submenu.GetSlot() != "fileTree.context" || submenu.GetSubmenu() != "ledger.new" || submenu.GetTitle() != "New ledger" ||
		submenu.GetIcon() != "plus" || submenu.GetGroup() != "0-new" || submenu.GetWhen() != "isFolder && !isMultiple" {
		t.Errorf("the submenu came out as %v", submenu)
	}
}
