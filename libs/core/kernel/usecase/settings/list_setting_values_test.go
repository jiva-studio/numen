package settings_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func key(path ...string) string { return settingsdomain.JoinPath(path) }

// A setting holds the stored value, else the default its plugin declares; a plugin with no stored
// flag is enabled; the connections section joins the app settings.
func TestListSettingValuesListsTheValueInForceOfEverySetting(t *testing.T) {
	file := &memoryFile{current: stored{
		values: map[string]any{
			"app.appearance.theme":    "preset:night",
			"org.example.scans.depth": 4.0,
			"org.example.off.enabled": false,
		},
		connections: `[{"id":"c"}]`,
	}}
	plugins := pluginInfos{
		{ID: "org.example.scans", Settings: []plugin.Setting{{Key: "depth"}, {Key: "mode", DefaultJSON: `"fast"`}, {Key: "none"}}},
		{ID: "org.example.off"},
	}
	app := appDeclarations{{Key: "appearance.theme"}, {Key: "appearance.mode"}}

	got, err := settings.NewListSettingValues(jsonCodec{}, file, file, app, plugins).Execute()

	want := settingsdomain.Values{
		key("app", "appearance", "theme"):   `"preset:night"`,
		key("org.example.scans", "depth"):   `4`,
		key("org.example.scans", "mode"):    `"fast"`,
		key("org.example.scans", "enabled"): `true`,
		key("org.example.off", "enabled"):   `false`,
		key("connections"):                  `[{"id":"c"}]`,
	}
	if err != nil || len(got) != len(want) {
		t.Fatalf("values = %v, %v", got, err)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%q = %q, want %q", k, got[k], v)
		}
	}
}

// The values of an unconfigured installation come from the defaults, not from the file.
func TestListSettingValuesListsTheDefaultsOfAnUnconfiguredInstallation(t *testing.T) {
	file := &memoryFile{current: stored{values: map[string]any{"app.appearance.theme": "preset:night"}}}
	list := settings.NewListSettingValues(jsonCodec{}, file, file, appDeclarations{{Key: "appearance.theme"}}, none.PluginInfos{})

	got, err := list.ListDefaults()

	if err != nil || len(got) != 0 {
		t.Fatalf("values = %v, %v", got, err)
	}
}
