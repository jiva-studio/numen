package plugin_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func newScanner() manifest.Manifest {
	return manifest.Manifest{
		ID: "org.example.scanner",
		Settings: []manifest.Setting{
			{Key: "provider", Type: "enum", Title: "Provider", Default: "remote",
				Options: manifest.SettingOptions{{Value: "remote"}, {Value: "local"}}},
			{Key: "model", Type: "enum", Title: "Model", OptionsBy: &manifest.OptionsBy{
				Setting: "provider",
				Options: map[string]manifest.SettingOptions{
					"remote": {{Value: "big"}, {Value: "huge"}},
					"local":  {{Value: "small"}},
				},
			}},
			{Key: "url", Type: "string", Title: "URL", Default: "https://example.org",
				When: &manifest.Condition{Setting: "provider", Is: []string{"remote"}}},
		},
	}
}

func listValues(options manifest.SettingOptions) []string {
	var values []string
	for _, one := range options {
		values = append(values, one.Value)
	}
	return values
}

func TestSettingsAreListedInTheOrderOfTheirDeclaration(t *testing.T) {
	settings := plugin.ListSettings(newScanner(), map[string]any{})

	var keys []string
	for _, one := range settings {
		keys = append(keys, one.Key)
	}
	if want := []string{"provider", "model", "url"}; !slices.Equal(keys, want) {
		t.Errorf("keys = %v, want %v", keys, want)
	}
}

func TestASettingCarriesTheOptionsInForceForTheValues(t *testing.T) {
	declared := newScanner()

	remote := plugin.ListSettings(declared, map[string]any{"provider": "remote"})
	local := plugin.ListSettings(declared, map[string]any{"provider": "local"})

	if got := listValues(remote[1].ActiveOptions); !slices.Equal(got, []string{"big", "huge"}) {
		t.Errorf("options under the remote provider = %v", got)
	}
	if got := listValues(local[1].ActiveOptions); !slices.Equal(got, []string{"small"}) {
		t.Errorf("options under the local provider = %v", got)
	}
}

func TestASettingWithoutOptionsByKeepsItsDeclaredOptions(t *testing.T) {
	settings := plugin.ListSettings(newScanner(), map[string]any{"provider": "local"})

	if got := listValues(settings[0].ActiveOptions); !slices.Equal(got, []string{"remote", "local"}) {
		t.Errorf("options = %v, want the declared ones", got)
	}
}

func TestSettingVisibilityFollowsItsCondition(t *testing.T) {
	declared := newScanner()

	remote := plugin.ListSettings(declared, map[string]any{"provider": "remote"})
	local := plugin.ListSettings(declared, map[string]any{"provider": "local"})

	if !remote[2].IsShown || local[2].IsShown {
		t.Errorf("url shown under remote = %v, under local = %v, want true and false", remote[2].IsShown, local[2].IsShown)
	}
	if !local[0].IsShown {
		t.Error("a setting with no condition is hidden")
	}
}

func TestASettingCarriesItsDefaultAndItsValueAsJSON(t *testing.T) {
	settings := plugin.ListSettings(newScanner(), map[string]any{"provider": "local", "model": "small"})

	if settings[0].DefaultJSON != `"remote"` {
		t.Errorf("default = %s, want \"remote\"", settings[0].DefaultJSON)
	}
	if settings[1].DefaultJSON != "" {
		t.Errorf("default of a setting with none = %s, want empty", settings[1].DefaultJSON)
	}
	if settings[0].ValueJSON != `"local"` || settings[1].ValueJSON != `"small"` {
		t.Errorf("values = %s and %s, want \"local\" and \"small\"", settings[0].ValueJSON, settings[1].ValueJSON)
	}
	if settings[2].ValueJSON != "" {
		t.Errorf("value of a setting that holds none = %s, want empty", settings[2].ValueJSON)
	}
}

func TestSettingValuesAreFormattedAsTextOrJSON(t *testing.T) {
	got := plugin.FormatSettingValues(map[string]any{
		"name":  "scans",
		"depth": 3,
		"flag":  true,
		"tags":  []string{"a", "b"},
	})

	want := map[string]string{"name": "scans", "depth": "3", "flag": "true", "tags": `["a","b"]`}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %q, want %q", key, got[key], value)
		}
	}
	if len(got) != len(want) {
		t.Errorf("formatted %d values, want %d", len(got), len(want))
	}
}

func TestAValueThatCannotBeEncodedIsFormattedWithItsPrintedForm(t *testing.T) {
	got := plugin.FormatSettingValues(map[string]any{"channel": make(chan int)})

	if got["channel"] == "" {
		t.Error("a value JSON cannot encode was left out")
	}
}
