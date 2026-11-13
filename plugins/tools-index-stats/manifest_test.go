package indexstats

import (
	"encoding/json"
	"slices"
	"testing"
)

type manifestShape struct {
	Targets     []string `json:"targets"`
	Permissions []string `json:"permissions"`
	Services    []string `json:"services"`
	Contributes struct {
		Slots []struct {
			Slot string `json:"slot"`
			View string `json:"view"`
		} `json:"slots"`
	} `json:"contributes"`
	Settings []json.RawMessage `json:"settings"`
}

func readManifest(t *testing.T) manifestShape {
	t.Helper()
	var shape manifestShape
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}
	return shape
}

// The plugin runs in the editor, reads the search index, follows the changes of the vault, and draws one view in the title bar.
func TestTheManifestDeclaresTheStatusViewTheTargetAndThePermission(t *testing.T) {
	shape := readManifest(t)

	if !slices.Equal(shape.Targets, []string{"desktop.editor"}) {
		t.Errorf("targets = %v", shape.Targets)
	}
	if !slices.Equal(shape.Permissions, []string{"index:read", "vault:read"}) {
		t.Errorf("permissions = %v", shape.Permissions)
	}
	if !slices.Equal(shape.Services, []string{"md.numen.tools.index.stats"}) {
		t.Errorf("services = %v", shape.Services)
	}
	if len(shape.Contributes.Slots) != 1 || shape.Contributes.Slots[0].Slot != "titleBar.status" || shape.Contributes.Slots[0].View != "stats" {
		t.Errorf("slots = %+v", shape.Contributes.Slots)
	}
}

// The plugin has no settings: the title bar item is part of the editor and needs none.
func TestTheManifestDeclaresNoSettings(t *testing.T) {
	if got := readManifest(t).Settings; len(got) != 0 {
		t.Errorf("settings = %s", got)
	}
}
