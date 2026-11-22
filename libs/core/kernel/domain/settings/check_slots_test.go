package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestSlotLayoutAccepted(t *testing.T) {
	declared := manifest.Setting{Key: "slots", Type: manifest.SettingTypeSlots}
	layout := map[string]any{
		"plexGraph.nodeBelow": []any{
			map[string]any{"id": "a/view/headings"},
			map[string]any{"id": "b/view/related", "hidden": true},
			map[string]any{"id": "---"},
		},
	}
	if err := settings.CheckDeclared(declared, nil, layout); err != nil {
		t.Fatalf("a layout was refused: %v", err)
	}
}

func TestSlotLayoutRefused(t *testing.T) {
	declared := manifest.Setting{Key: "slots", Type: manifest.SettingTypeSlots}
	for name, value := range map[string]any{
		"text":        "x",
		"list":        map[string]any{"s": "x"},
		"entry":       map[string]any{"s": []any{"x"}},
		"id missing":  map[string]any{"s": []any{map[string]any{"hidden": true}}},
		"id empty":    map[string]any{"s": []any{map[string]any{"id": ""}}},
		"hidden text": map[string]any{"s": []any{map[string]any{"id": "a", "hidden": "yes"}}},
	} {
		if err := settings.CheckDeclared(declared, nil, value); !errors.Is(err, settings.ErrWrongType) {
			t.Errorf("%s gave %v", name, err)
		}
	}
}
