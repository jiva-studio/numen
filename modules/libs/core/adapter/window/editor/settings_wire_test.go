package editor

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

func TestTheProgramWhoseModelsASettingIsOfferedReachesTheWire(t *testing.T) {
	wire := newWirePluginInfos([]settings.PluginInfo{{
		ID: "a.b",
		Settings: []settings.PluginSetting{
			{Key: "model", Type: "string", ModelsOf: "prog"},
			{Key: "other", Type: "string"},
		},
	}})

	got := wire[0].GetSettings()
	if got[0].GetModelsOf() != "prog" || got[1].GetModelsOf() != "" {
		t.Errorf("the settings carry %q and %q", got[0].GetModelsOf(), got[1].GetModelsOf())
	}
}
