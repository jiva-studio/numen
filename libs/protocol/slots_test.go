package protocol

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func TestTheSlotsAreTheOnesSlotsJSONLists(t *testing.T) {
	text, err := os.ReadFile("slots.json")
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
		if !IsSlot(one.Name) || IsWindowSlot(one.Name) != one.IsWindow {
			t.Errorf("slot %q: IsSlot %v, IsWindowSlot %v", one.Name, IsSlot(one.Name), IsWindowSlot(one.Name))
		}
	}
	if !slices.Equal(names, Slots) {
		t.Errorf("Slots = %v, want %v", Slots, names)
	}
	if IsSlot("sideBar") || IsWindowSlot("titleBar.commands") {
		t.Error("a name outside the list, or an inner slot, is taken for a slot of the window")
	}
}
