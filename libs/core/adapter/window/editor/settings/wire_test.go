package settings

import (
	"testing"

	setting "github.com/jiva-studio/numen/libs/core/usecase/settings"
)

func TestNewWirePluginInfos_KeepsModelsOfProgram(t *testing.T) {
	wire := NewWirePluginInfos([]setting.PluginInfo{{
		ID: "a.b",
		Settings: []setting.PluginSetting{
			{Key: "model", Type: "string", ModelsOf: "prog", Icon: "audio-lines"},
			{Key: "other", Type: "string"},
		},
	}})

	got := wire[0].GetSettings()
	if got[0].GetModelsOf() != "prog" || got[0].GetIcon() != "audio-lines" || got[1].GetModelsOf() != "" || got[1].GetIcon() != "" {
		t.Errorf("the settings carry %v and %v", got[0], got[1])
	}
}

func TestNewWirePluginInfos_CarriesSettingsViews(t *testing.T) {
	wire := NewWirePluginInfos([]setting.PluginInfo{{
		ID:            "a.b",
		SettingsViews: []setting.PluginSettingsView{{ID: "floor", Title: "Search relevance", After: "model"}},
	}})

	got := wire[0].GetSettingsViews()
	if len(got) != 1 || got[0].GetId() != "floor" || got[0].GetTitle() != "Search relevance" || got[0].GetAfter() != "model" {
		t.Errorf("the settings views carry %v", got)
	}
}
