package manifest_test

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestTheSlotsAreTheOnesSlotsJSONLists(t *testing.T) {
	text, err := os.ReadFile("../../../core/protocol/slots.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Slots []struct {
			Name     string `json:"name"`
			IsWindow bool   `json:"isWindow"`
		} `json:"slots"`
	}
	if err := json.Unmarshal(text, &document); err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, one := range document.Slots {
		names = append(names, one.Name)
		if !manifest.IsSlot(one.Name) || manifest.IsWindowSlot(one.Name) != one.IsWindow {
			t.Errorf("slot %q: IsSlot %v, IsWindowSlot %v", one.Name, manifest.IsSlot(one.Name), manifest.IsWindowSlot(one.Name))
		}
	}
	if !slices.Equal(names, manifest.Slots) {
		t.Errorf("Slots = %v, want %v", manifest.Slots, names)
	}
	if manifest.IsSlot("sideBar") || manifest.IsWindowSlot("titleBar.commands") {
		t.Error("a name outside the list, or an inner slot, is taken for a slot of the window")
	}
}
