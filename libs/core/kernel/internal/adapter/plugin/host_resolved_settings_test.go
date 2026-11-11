package plugin_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func newResolvedHost(t *testing.T) *container.PluginRoles {
	t.Helper()
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "sample")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.sample", "name": "Sample", "version": "1.0.0", "targets": ["desktop.editor"],
		"settings": [
			{"key": "provider", "type": "enum", "title": "Provider", "default": "remote",
				"options": [{"value": "remote"}, {"value": "local"}]},
			{"key": "model", "type": "enum", "title": "Model",
				"optionsBy": {"setting": "provider", "options": {
					"remote": [{"value": "big"}, {"value": "huge"}],
					"local": [{"value": "small"}]}}},
			{"key": "url", "type": "string", "title": "URL", "default": "https://example.org",
				"when": {"setting": "provider", "is": ["remote"]}}
		]
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	return host
}

func findInfoSetting(t *testing.T, host *container.PluginRoles, key string) plugindomain.Setting {
	t.Helper()
	for _, info := range host.Registry.GetPluginInfos() {
		for _, one := range info.Settings {
			if one.Key == key {
				return one
			}
		}
	}
	t.Fatalf("no setting %s", key)
	return plugindomain.Setting{}
}

func listValues(options manifest.SettingOptions) []string {
	var values []string
	for _, one := range options {
		values = append(values, one.Value)
	}
	return values
}

func TestThePluginInfoCarriesTheOptionsInForceForTheStoredValues(t *testing.T) {
	host := newResolvedHost(t)
	if got := listValues(findInfoSetting(t, host, "model").ActiveOptions); !slices.Equal(got, []string{"big", "huge"}) {
		t.Errorf("the options under the remote provider are %v", got)
	}

	if err := updateSetting(t.Context(), host, "md.numen.sample", "provider", "local"); err != nil {
		t.Fatal(err)
	}
	if got := listValues(findInfoSetting(t, host, "model").ActiveOptions); !slices.Equal(got, []string{"small"}) {
		t.Errorf("the options under the local provider are %v", got)
	}
}

func TestThePluginInfoCarriesTheValueInForceOfASettingThatFollowsAnother(t *testing.T) {
	host := newResolvedHost(t)
	if got := findInfoSetting(t, host, "model").ValueJSON; got != `"big"` {
		t.Errorf("the model in force is %s, want the first option of the provider", got)
	}
	if got := findInfoSetting(t, host, "url").ValueJSON; got != `"https://example.org"` {
		t.Errorf("the url in force is %s, want its default", got)
	}
	if err := updateSetting(t.Context(), host, "md.numen.sample", "model", "huge"); err != nil {
		t.Fatal(err)
	}
	if got := findInfoSetting(t, host, "model").ValueJSON; got != `"huge"` {
		t.Errorf("the model in force is %s, want the stored one", got)
	}
}

func TestThePluginInfoMarksTheSettingsTheirConditionsShow(t *testing.T) {
	host := newResolvedHost(t)
	if !findInfoSetting(t, host, "url").IsShown {
		t.Error("the url is hidden under the remote provider")
	}
	if !findInfoSetting(t, host, "provider").IsShown {
		t.Error("a setting with no condition is hidden")
	}
	if err := updateSetting(t.Context(), host, "md.numen.sample", "provider", "local"); err != nil {
		t.Fatal(err)
	}
	if findInfoSetting(t, host, "url").IsShown {
		t.Error("the url is shown under the local provider")
	}
}
