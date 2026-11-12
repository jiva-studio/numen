package plugin

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestASettingOfRowsIsShownWithItsOperationsAndItsRowLimit(t *testing.T) {
	limit := 2.0
	declared := manifest.Setting{Key: "connections", Type: manifest.SettingTypeConnections, Operations: []string{"llm.embed", "llm.profile"}, Max: &limit}

	view := NewSetting(declared, "", nil)

	if !slices.Equal(view.Operations, []string{"llm.embed", "llm.profile"}) || view.Max == nil || *view.Max != 2 {
		t.Errorf("the setting shown = %+v, want operations llm.embed and llm.profile and max 2", view)
	}
}
