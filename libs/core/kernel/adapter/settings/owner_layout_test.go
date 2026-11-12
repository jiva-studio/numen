package settings_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
)

// handWritten is a file in the layout by owner: the version at the root, the shell under app, and one
// object per plugin id, with `enabled` beside the settings of the plugin.
const handWritten = `{
  "v": 1,
  "app": {
    "appearance": {"theme": "mine:dracula", "mode": "dark", "interfaceScale": 1.25, "textScale": 1.5},
    "naming": {"syncTitleAndFilename": false},
    "performance": {"profile": "eco"}
  },
  "md.numen.sample.speech": {
    "enabled": true,
    "model": "small",
    "maxTokens": 256
  },
  "md.numen.sample.notes": {
    "enabled": false
  }
}
`

func readRoot(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("%s: %v", raw, err)
	}
	return root
}

func listKeys(root map[string]json.RawMessage) string {
	keys := make([]string, 0, len(root))
	for key := range root {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return strings.Join(keys, ", ")
}

func TestAFileInTheLayoutByOwnerIsRead(t *testing.T) {
	cfg, err := openAt(write(t, handWritten))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]any{
		"app.appearance.theme":             "mine:dracula",
		"app.appearance.mode":              "dark",
		"app.appearance.interfaceScale":    1.25,
		"app.appearance.textScale":         1.5,
		"app.naming.syncTitleAndFilename":  false,
		"app.performance.profile":          "eco",
		"md.numen.sample.speech/model":     "small",
		"md.numen.sample.speech/maxTokens": 256.0,
		"md.numen.sample.speech/enabled":   true,
		"md.numen.sample.notes/enabled":    false,
	} {
		var path []string
		if owner, key, isPlugin := strings.Cut(name, "/"); isPlugin {
			path = []string{owner, key}
		} else {
			path = strings.Split(name, ".")
		}
		got, ok := cfg.GetValue(path...)
		if !ok || got != want {
			t.Errorf("%v is %v (found %v), want %v", path, got, ok, want)
		}
	}
}

func TestTheVersionIsAtTheRootOfTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen", "numen.json")
	if _, err := openAt(path); err != nil {
		t.Fatal(err)
	}
	root := readRoot(t, path)
	if string(root["v"]) != "1" {
		t.Errorf("the version at the root is %s", root["v"])
	}
	var app map[string]json.RawMessage
	if err := json.Unmarshal(root["app"], &app); err != nil {
		t.Fatalf("app is %q", root["app"])
	}
	if _, nested := app["v"]; nested {
		t.Error("the version is under app")
	}
}

func TestAnUnconfiguredInstallationWritesAnAppObjectAndNoSectionAtTheRoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen", "numen.json")
	if _, err := openAt(path); err != nil {
		t.Fatal(err)
	}
	root := readRoot(t, path)
	if _, ok := root["app"]; !ok {
		t.Errorf("the file has no app object: %s", listKeys(root))
	}
	for _, section := range []string{"appearance", "naming", "performance", "agent", "plugins"} {
		if _, ok := root[section]; ok {
			t.Errorf("the section %s is at the root of the file", section)
		}
	}
}

func TestTheFileHasNoPluginsObjectWhenPluginsHaveSettings(t *testing.T) {
	cfg, err := openAt(write(t, handWritten))
	if err != nil {
		t.Fatal(err)
	}
	content, err := cfg.WriteJSON()
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &root); err != nil {
		t.Fatal(err)
	}
	if _, ok := root["plugins"]; ok {
		t.Errorf("the settings carry a plugins object:\n%s", content)
	}
	for _, id := range []string{"app", "md.numen.sample.speech", "md.numen.sample.notes"} {
		if _, ok := root[id]; !ok {
			t.Errorf("the settings carry no object for %s:\n%s", id, content)
		}
	}
	if string(root["v"]) != "1" {
		t.Errorf("the version at the root is %s", root["v"])
	}
}

func TestAPluginObjectHasEnabledAmongItsSettings(t *testing.T) {
	cfg, err := openAt(write(t, handWritten))
	if err != nil {
		t.Fatal(err)
	}
	owner := cfg.GetOwner("md.numen.sample.speech")
	if owner["enabled"] != true || owner["model"] != "small" {
		t.Errorf("the object of the plugin is %v", owner)
	}
}

func TestAPathIsNotJoinedWhereAPluginIdHasDots(t *testing.T) {
	path := write(t, `{"v": 1}`)
	err := settings.NewFiles().Save(path, newDocument(),
		settings.Setting{At: []string{"md.numen.sample.speech", "model"}, Written: "small"},
		settings.Setting{At: []string{"md.numen.sample.speech", "enabled"}, Written: true})
	if err != nil {
		t.Fatal(err)
	}
	root := readRoot(t, path)
	var plugin map[string]any
	if err := json.Unmarshal(root["md.numen.sample.speech"], &plugin); err != nil {
		t.Fatalf("the object of the plugin id is missing: %s", listKeys(root))
	}
	if plugin["model"] != "small" || plugin["enabled"] != true {
		t.Errorf("the object of the plugin is %v", plugin)
	}
	if _, found := root["md"]; found {
		t.Errorf("the id was split at its dots: %s", listKeys(root))
	}
}

func TestAPluginSettingIsStoredInTheObjectOfThePlugin(t *testing.T) {
	path := write(t, handWritten)
	err := settings.NewFiles().Save(path, newDocument(),
		settings.Setting{At: []string{"md.numen.sample.speech", "model"}, Written: "large"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(handWritten, `"model": "small"`, `"model": "large"`, 1)
	if string(raw) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", raw, want)
	}
}

func TestADefaultSettingOfTheShellIsInForceWhereTheFileNamesNone(t *testing.T) {
	cfg, err := openAt(write(t, `{"v": 1, "md.numen.sample.speech": {"enabled": true}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := cfg.GetValue("app", "appearance", "theme"); !ok || got != settings.DefaultTheme {
		t.Errorf("the theme in force is %v (found %v)", got, ok)
	}
	if got, ok := cfg.GetValue("app", "naming", "syncTitleAndFilename"); !ok || got != true {
		t.Errorf("the naming in force is %v (found %v)", got, ok)
	}
}
