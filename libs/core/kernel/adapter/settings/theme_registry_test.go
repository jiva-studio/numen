package settings_test

import (
	"slices"
	"testing"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
)

func TestTheThemeRegistryOffersTheThemesAndKnowsNoOtherRegistry(t *testing.T) {
	registry := settingsadapter.ThemeRegistry(func() []themedomain.Theme {
		return []themedomain.Theme{{Name: "dusk", Title: "Dusk"}, {Name: "dawn", Title: "Dawn"}}
	})

	options, found := registry(settingsdomain.RegistryThemes)
	if !found || len(options) != 2 || options[0].Value != "dusk" || options[1].Title != "Dawn" {
		t.Fatalf("options = %+v, found %v", options, found)
	}
	if names, found := registry.GetNames()(settingsdomain.RegistryThemes); !found || !slices.Equal(names, []string{"dusk", "dawn"}) {
		t.Errorf("names = %v, found %v", names, found)
	}
	if _, found := registry("fonts"); found {
		t.Error("an unknown registry was found")
	}
}
