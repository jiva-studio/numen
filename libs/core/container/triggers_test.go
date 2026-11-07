package container

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func TestCreateTriggerConfigNestsTheSettingsByOwnerAndKeyAsTheWindowDoes(t *testing.T) {
	settingsFile := `{"app": {"editor.lineNumbers": true}, "org.example.scans": {"isAutomatic": true}}`
	app := []pluginport.PluginSetting{
		{Key: "editor.lineNumbers", DefaultJSON: "false"},
		{Key: "appearance", DefaultJSON: `"dark"`},
	}
	plugins := []pluginport.PluginInfo{
		{ID: "org.example.scans", Settings: []pluginport.PluginSetting{
			{Key: "isAutomatic", DefaultJSON: "false"},
			{Key: "dpi", DefaultJSON: `"300"`},
		}},
	}

	got, err := createTriggerConfig(settingsFile, app, plugins)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"app": map[string]any{"editor": map[string]any{"lineNumbers": true}, "appearance": "dark"},
		"org": map[string]any{"example": map[string]any{"scans": map[string]any{"isAutomatic": true, "dpi": "300"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("config = %v, want %v", got, want)
	}
}

func TestCreateTriggerConfigRejectsASettingsFileThatIsNotJSON(t *testing.T) {
	if _, err := createTriggerConfig("{", nil, nil); err == nil {
		t.Error("a settings file that is no JSON made a config")
	}
}

func TestCreateTriggerConfigReadsTheSettingsFileAsTheCoreWritesIt(t *testing.T) {
	settingsFile := `{"v": 1, "app": {"appearance": {"theme": "preset:paper"}}, "org.example.scans": {"isAutomatic": true}}`
	app := []pluginport.PluginSetting{{Key: "appearance.theme", DefaultJSON: `"preset:numen"`}}
	plugins := []pluginport.PluginInfo{
		{ID: "org.example.scans", Settings: []pluginport.PluginSetting{{Key: "isAutomatic", DefaultJSON: "false"}}},
	}

	got, err := createTriggerConfig(settingsFile, app, plugins)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"app": map[string]any{"appearance": map[string]any{"theme": "preset:paper"}},
		"org": map[string]any{"example": map[string]any{"scans": map[string]any{"isAutomatic": true}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("config = %v, want %v", got, want)
	}
}
