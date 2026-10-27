package serviceproxy

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func TestTheFramesOfAPluginAreListedWhileItIsEnabled(t *testing.T) {
	host := &servingHost{
		infos: []pluginport.PluginInfo{{ID: "org.example.off"}, {ID: "org.example.player"}, {ID: "org.example.reader"}},
		manifests: map[string]pluginport.PluginManifest{
			"org.example.player": {ID: "org.example.player", Frames: []string{"https://player.example"}},
			"org.example.reader": {ID: "org.example.reader", Frames: []string{"https://player.example", "https://maps.example"}},
		},
	}
	proxy := Open(host)

	if got := proxy.ListFrameOrigins(); !slices.Equal(got, []string{"https://player.example", "https://maps.example"}) {
		t.Errorf("the window frames %v", got)
	}
	if got := proxy.GetFrameOrigins("org.example.off"); len(got) != 0 {
		t.Errorf("a plugin that is not enabled frames %v", got)
	}
}
