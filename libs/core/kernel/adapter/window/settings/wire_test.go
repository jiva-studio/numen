package settings

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestNewWirePluginInfos_CarriesTheIcon(t *testing.T) {
	wire := NewWirePluginInfos([]plugin.Info{{
		ID: "a.b",
		Settings: []plugin.Setting{
			{Key: "model", Type: "string", Icon: "audio-lines"},
			{Key: "other", Type: "string"},
		},
	}})

	got := wire[0].GetSettings()
	if got[0].GetIcon() != "audio-lines" || got[1].GetIcon() != "" {
		t.Errorf("the settings carry %v and %v", got[0], got[1])
	}
}

func TestNewWirePluginInfos_CarriesSettingsViews(t *testing.T) {
	wire := NewWirePluginInfos([]plugin.Info{{
		ID:            "a.b",
		SettingsViews: []manifest.SettingsView{{ID: "floor", Title: "Search relevance"}},
	}})

	got := wire[0].GetSettingsViews()
	if len(got) != 1 || got[0].GetId() != "floor" || got[0].GetTitle() != "Search relevance" {
		t.Errorf("the settings views carry %v", got)
	}
}

func TestNewWirePluginInfos_CarriesIsOffByDefault(t *testing.T) {
	wire := NewWirePluginInfos([]plugin.Info{{ID: "a.b", IsOffByDefault: true}, {ID: "c.d"}})

	if !wire[0].GetIsOffByDefault() || wire[1].GetIsOffByDefault() {
		t.Errorf("the plugins carry is_off_by_default %v and %v", wire[0].GetIsOffByDefault(), wire[1].GetIsOffByDefault())
	}
}

func TestNewWirePluginInfos_CarriesMultiple(t *testing.T) {
	wire := NewWirePluginInfos([]plugin.Info{{
		ID: "a.b",
		Settings: []plugin.Setting{
			{Key: "programs", Type: "file", Multiple: true},
			{Key: "dictionary", Type: "file"},
		},
	}})

	got := wire[0].GetSettings()
	if !got[0].GetMultiple() || got[1].GetMultiple() {
		t.Errorf("the settings carry %v and %v", got[0], got[1])
	}
}

func TestNewWirePluginInfos_CarriesWhatTheCoreResolved(t *testing.T) {
	wire := NewWirePluginInfos([]plugin.Info{{
		ID: "a.b",
		Settings: []plugin.Setting{{
			Key:           "model",
			Type:          "enum",
			ActiveOptions: manifest.SettingOptions{{Value: "big", Title: "Big"}},
			IsShown:       true,
			ValueJSON:     `"big"`,
		}, {Key: "url", Type: "string"}},
	}})

	got := wire[0].GetSettings()
	if len(got[0].GetActiveOptions()) != 1 || got[0].GetActiveOptions()[0].GetValue() != "big" {
		t.Errorf("the options in force are %v", got[0].GetActiveOptions())
	}
	if !got[0].GetIsShown() || got[0].GetValueJson() != `"big"` {
		t.Errorf("the setting carries shown=%v and value %q", got[0].GetIsShown(), got[0].GetValueJson())
	}
	if got[1].GetIsShown() || got[1].GetValueJson() != "" {
		t.Errorf("a setting resolved to nothing carries shown=%v and value %q", got[1].GetIsShown(), got[1].GetValueJson())
	}
}

func TestNewWirePluginInfos_CarriesThePreviewFlagAndTheRegistry(t *testing.T) {
	wire := NewWirePluginInfos([]plugin.Info{{
		ID: "a.b",
		Settings: []plugin.Setting{
			{Key: "theme", Type: "enum", Preview: true, OptionsFrom: "themes"},
			{Key: "other", Type: "string"},
		},
	}})

	got := wire[0].GetSettings()
	if !got[0].GetPreview() || got[0].GetOptionsFrom() != "themes" {
		t.Errorf("the theme carries preview=%v from %q", got[0].GetPreview(), got[0].GetOptionsFrom())
	}
	if got[1].GetPreview() || got[1].GetOptionsFrom() != "" {
		t.Errorf("a plain setting carries preview=%v from %q", got[1].GetPreview(), got[1].GetOptionsFrom())
	}
}

func TestNewWirePluginInfos_CarriesTheOperationsAndTheRowLimitOfASettingOfRows(t *testing.T) {
	limit := 3.0
	wire := NewWirePluginInfos([]plugin.Info{{
		ID: "a.b",
		Settings: []plugin.Setting{
			{Key: "connections", Type: "connections", Operations: []string{"llm.complete"}, Max: &limit},
			{Key: "other", Type: "string"},
		},
	}})

	got := wire[0].GetSettings()
	if !slices.Equal(got[0].GetOperations(), []string{"llm.complete"}) || got[0].GetMax() != 3 {
		t.Errorf("the rows carry operations %q and max %v", got[0].GetOperations(), got[0].GetMax())
	}
	if len(got[1].GetOperations()) != 0 {
		t.Errorf("a plain setting carries operations %q", got[1].GetOperations())
	}
}
