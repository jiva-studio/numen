package pluginport

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

func TestASettingOfRowsIsShownWithItsOperationAndItsRowLimit(t *testing.T) {
	limit := 2.0
	declared := manifest.Setting{Key: "connections", Type: manifest.SettingTypeConnections, Operation: "embed", Max: &limit}

	shown := NewPluginSetting(declared, "", nil)

	if shown.Operation != "embed" || shown.Max == nil || *shown.Max != 2 {
		t.Errorf("the setting shown = %+v, want operation embed and max 2", shown)
	}
}
