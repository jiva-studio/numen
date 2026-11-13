package servicebridge

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestTheFramesOfAPluginAreListedWhileItIsEnabled(t *testing.T) {
	host := &servingHost{
		infos: []plugin.Info{{ID: "org.example.off"}, {ID: "org.example.player"}, {ID: "org.example.reader"}},
		manifests: map[string]manifest.Manifest{
			"org.example.player": {ID: "org.example.player", Contributes: manifest.Contributes{FrameOrigins: []string{"https://player.example"}}},
			"org.example.reader": {ID: "org.example.reader", Contributes: manifest.Contributes{FrameOrigins: []string{"https://player.example", "https://maps.example"}}},
		},
	}
	proxy := Open(host, host)

	if got := proxy.ListFrameOrigins(); !slices.Equal(got, []string{"https://player.example", "https://maps.example"}) {
		t.Errorf("the window frames %v", got)
	}
	if got := proxy.GetFrameOrigins("org.example.off"); len(got) != 0 {
		t.Errorf("a plugin that is not enabled frames %v", got)
	}
}
