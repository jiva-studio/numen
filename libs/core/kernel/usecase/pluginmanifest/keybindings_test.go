package pluginmanifest

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestListKeybindings_ReadsTheManifestsOfEnabledPluginsInOrder(t *testing.T) {
	host := &manifestHost{
		plugins: []manifest.Discovered{
			{Manifest: manifest.Manifest{ID: "org.example.atlas", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{Keybindings: []manifest.Keybinding{
				{Key: "mod+shift+a", Command: "atlas.open"},
				{Key: "a", Command: "atlas.open", When: "!window.isFieldFocused"},
			}}}},
			{Manifest: manifest.Manifest{ID: "org.example.off", Frontend: &manifest.Frontend{}, Contributes: manifest.Contributes{Keybindings: []manifest.Keybinding{
				{Key: "o", Command: "off.run"},
			}}}},
		},
		disabled: map[string]bool{"org.example.off": true},
	}

	entries := New(host).ListKeybindings()

	if len(entries) != 2 {
		t.Fatalf("got %v, want the two keybindings of the enabled plugin", entries)
	}
	if entries[0].PluginID != "org.example.atlas" || entries[0].Keybinding.Key != "mod+shift+a" || entries[1].Keybinding.When != "!window.isFieldFocused" {
		t.Errorf("got %v", entries)
	}
}
