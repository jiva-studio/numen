package settings

import (
	"testing"

	setting "github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

func TestTheProgramWhoseModelsASettingIsOfferedReachesTheWire(t *testing.T) {
	wire := NewWirePluginInfos([]setting.PluginInfo{{
		ID: "a.b",
		Settings: []setting.PluginSetting{
			{Key: "model", Type: "string", ModelsOf: "prog"},
			{Key: "other", Type: "string"},
		},
	}})

	got := wire[0].GetSettings()
	if got[0].GetModelsOf() != "prog" || got[1].GetModelsOf() != "" {
		t.Errorf("the settings carry %q and %q", got[0].GetModelsOf(), got[1].GetModelsOf())
	}
}
