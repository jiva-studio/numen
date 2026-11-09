package whenconfig_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/whenconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

func TestTheConfigOfAConditionIsTheResolvedValueOrTheDefaultOfEachSetting(t *testing.T) {
	app := []pluginport.PluginSetting{
		{Key: "editor.lineNumbers", DefaultJSON: "false", ValueJSON: "true"},
		{Key: "theme", DefaultJSON: `"dusk"`},
		{Key: "nothing"},
	}
	plugins := []pluginport.PluginInfo{{ID: "org.example.scans", Settings: []pluginport.PluginSetting{
		{Key: "isAutomatic", DefaultJSON: "false", ValueJSON: "true"},
		{Key: "pages", DefaultJSON: "3"},
	}}}

	got := whenconfig.New(app, plugins)

	want := map[string]any{
		"app": map[string]any{"editor": map[string]any{"lineNumbers": true}, "theme": "dusk"},
		"org": map[string]any{"example": map[string]any{"scans": map[string]any{"isAutomatic": true, "pages": float64(3)}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("config = %v, want %v", got, want)
	}
}
