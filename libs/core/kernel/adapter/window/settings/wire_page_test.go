package settings

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestNewWirePluginInfos_CarriesThePageOfASettingsView(t *testing.T) {
	wire := NewWirePluginInfos([]plugin.Info{{
		ID: "a.b",
		SettingsViews: []manifest.SettingsView{
			{ID: "vaults", Title: "Vaults"},
			{ID: "subscription", Title: "Subscription", Page: "identity"},
		},
	}})

	got := wire[0].GetSettingsViews()
	if len(got) != 2 {
		t.Fatalf("the settings views = %v, want 2", got)
	}
	if got[0].GetPage() != "" || got[1].GetPage() != "identity" {
		t.Errorf("the settings views carry the pages %q and %q, want none and identity", got[0].GetPage(), got[1].GetPage())
	}
}
