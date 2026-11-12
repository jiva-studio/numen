package pluginmanifest

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestListTriggers_ReadsTheTriggersOfTheEnabledPlugins(t *testing.T) {
	added := manifest.Trigger{Event: manifest.EventFileAdded, Command: "app:file.createText", When: "target.isFile"}
	host := &manifestHost{
		plugins: []manifest.Discovered{
			{Manifest: manifest.Manifest{ID: "org.example.scans", Contributes: manifest.Contributes{Triggers: []manifest.Trigger{added}}}},
			{Manifest: manifest.Manifest{ID: "org.example.off", Contributes: manifest.Contributes{Triggers: []manifest.Trigger{added}}}},
		},
		disabled: map[string]bool{"org.example.off": true},
	}

	got := New(host).ListTriggers()

	if want := []plugin.TriggerEntry{{PluginID: "org.example.scans", Trigger: added}}; !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
