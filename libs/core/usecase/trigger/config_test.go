package trigger_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/trigger"
)

func TestCreateConfigNestsTheSettingsByOwnerAndKeyAsTheWindowDoes(t *testing.T) {
	written := `{"app": {"editor.lineNumbers": true}, "org.example.scans": {"isAutomatic": true}}`
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

	got, err := trigger.CreateConfig(written, app, plugins)
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

func TestCreateConfigRejectsASettingsFileThatIsNotJSON(t *testing.T) {
	if _, err := trigger.CreateConfig("{", nil, nil); err == nil {
		t.Error("a settings file that is no JSON made a config")
	}
}
