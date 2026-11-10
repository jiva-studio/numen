package whenconfig_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/whenconfig"
)

func TestTheConfigOfAConditionIsTheResolvedValueOrTheDefaultOfEachSetting(t *testing.T) {
	app := []plugin.Setting{
		{Key: "editor.lineNumbers", DefaultJSON: "false", ValueJSON: "true"},
		{Key: "theme", DefaultJSON: `"dusk"`},
		{Key: "nothing"},
	}
	plugins := []plugin.Info{{ID: "org.example.scans", Settings: []plugin.Setting{
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
