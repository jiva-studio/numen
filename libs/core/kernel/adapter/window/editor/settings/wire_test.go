package settings

import (
	"testing"

	setting "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func TestNewWirePluginInfos_CarriesTheIcon(t *testing.T) {
	wire := NewWirePluginInfos([]setting.PluginInfo{{
		ID: "a.b",
		Settings: []setting.PluginSetting{
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
	wire := NewWirePluginInfos([]setting.PluginInfo{{
		ID:            "a.b",
		SettingsViews: []setting.PluginSettingsView{{ID: "floor", Title: "Search relevance"}},
	}})

	got := wire[0].GetSettingsViews()
	if len(got) != 1 || got[0].GetId() != "floor" || got[0].GetTitle() != "Search relevance" {
		t.Errorf("the settings views carry %v", got)
	}
}

func TestNewWirePluginInfos_CarriesMultiple(t *testing.T) {
	wire := NewWirePluginInfos([]setting.PluginInfo{{
		ID: "a.b",
		Settings: []setting.PluginSetting{
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
	wire := NewWirePluginInfos([]setting.PluginInfo{{
		ID: "a.b",
		Settings: []setting.PluginSetting{{
			Key:           "model",
			Type:          "enum",
			ActiveOptions: setting.PluginSettingOptions{{Value: "big", Title: "Big"}},
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
	wire := NewWirePluginInfos([]setting.PluginInfo{{
		ID: "a.b",
		Settings: []setting.PluginSetting{
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
