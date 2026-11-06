package commandbridge

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

func TestListTriggers_ReadsTheTriggersOfTheEnabledPlugins(t *testing.T) {
	added := manifest.Trigger{On: manifest.EventFileAdded, Command: "app:file.createText", When: "target.isFile"}
	host := &commandHost{
		plugins: []pluginport.DiscoveredPlugin{
			{Manifest: manifest.Manifest{ID: "org.example.scans", Contributes: manifest.Contributes{Triggers: []manifest.Trigger{added}}}},
			{Manifest: manifest.Manifest{ID: "org.example.off", Contributes: manifest.Contributes{Triggers: []manifest.Trigger{added}}}},
		},
		disabled: map[string]bool{"org.example.off": true},
	}

	got := Open(host).ListTriggers()

	if want := []pluginport.PluginTriggerEntry{{PluginID: "org.example.scans", Trigger: added}}; !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
