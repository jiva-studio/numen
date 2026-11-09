package manifest_test

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

func TestTheSlotsAreTheOnesSlotsJSONLists(t *testing.T) {
	text, err := os.ReadFile("../../protocol/slots.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Slots []struct {
			Name        string `json:"name"`
			IsWindow    bool   `json:"isWindow"`
			Arrangement string `json:"arrangement"`
		} `json:"slots"`
	}
	if err := json.Unmarshal(text, &document); err != nil {
		t.Fatal(err)
	}

	var names, userNames []string
	for _, one := range document.Slots {
		names = append(names, one.Name)
		isUser := one.Arrangement == "user"
		if isUser {
			userNames = append(userNames, one.Name)
		}
		if manifest.IsUserArrangedSlot(one.Name) != isUser {
			t.Errorf("slot %q: IsUserArrangedSlot %v, arrangement %q", one.Name, manifest.IsUserArrangedSlot(one.Name), one.Arrangement)
		}
		if !manifest.IsSlot(one.Name) || manifest.IsWindowSlot(one.Name) != one.IsWindow {
			t.Errorf("slot %q: IsSlot %v, IsWindowSlot %v", one.Name, manifest.IsSlot(one.Name), manifest.IsWindowSlot(one.Name))
		}
	}
	if !slices.Equal(names, manifest.Slots) {
		t.Errorf("Slots = %v, want %v", manifest.Slots, names)
	}
	if !slices.Equal(userNames, manifest.ListUserArrangedSlots()) {
		t.Errorf("ListUserArrangedSlots = %v, want %v", manifest.ListUserArrangedSlots(), userNames)
	}
	if manifest.IsUserArrangedSlot("sideBar") || manifest.IsUserArrangedSlot("fileTree.context") || manifest.IsUserArrangedSlot("titleBar") {
		t.Error("a name outside the list, or a slot the manifests fix, is taken for a slot the user arranges")
	}
	if manifest.IsSlot("sideBar") || manifest.IsWindowSlot("titleBar.commands") {
		t.Error("a name outside the list, or an inner slot, is taken for a slot of the window")
	}
}
