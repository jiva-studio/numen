package pluginmanifest

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestListSlots_ReadsTheManifestsOfEnabledPluginsInOrder(t *testing.T) {
	order := 5
	host := &manifestHost{
		plugins: []manifest.Discovered{
			{Manifest: manifest.Manifest{ID: "org.example.atlas", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{Slots: []manifest.SlotEntry{
				{Slot: "titleBar.commands", Command: "atlas.open", Group: "navigate", Order: &order},
				{Slot: "palette.searchResult", Command: "atlas.open"},
			}}}},
			{Manifest: manifest.Manifest{ID: "org.example.off", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{Slots: []manifest.SlotEntry{
				{Slot: "titleBar.commands", Command: "off.run"},
			}}}},
			{Manifest: manifest.Manifest{ID: "org.example.worker", Backend: manifest.Backend{Entrypoint: "bin/worker"}, Contributes: manifest.Contributes{Slots: []manifest.SlotEntry{
				{Slot: "titleBar.commands", Command: "app:note.create"},
			}}}},
		},
		disabled: map[string]bool{"org.example.off": true},
	}

	entries := New(host).ListSlots()

	if len(entries) != 3 {
		t.Fatalf("got %+v", entries)
	}
	if entries[0].PluginID != "org.example.atlas" || entries[0].Entry.Command != "atlas.open" || entries[0].Entry.Group != "navigate" ||
		entries[0].Entry.Order == nil || *entries[0].Entry.Order != 5 {
		t.Errorf("the first entry came out as %+v", entries[0])
	}
	if entries[1].PluginID != "org.example.atlas" || entries[1].Entry.Slot != "palette.searchResult" || entries[1].Entry.Order != nil {
		t.Errorf("the second entry came out as %+v", entries[1])
	}
	if entries[2].PluginID != "org.example.worker" || entries[2].Entry.Command != "app:note.create" {
		t.Errorf("the third entry came out as %+v", entries[2])
	}
}
