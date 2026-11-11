package updater

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
	Settings []struct {
		Key string `json:"key"`
	} `json:"settings"`
}

func readManifest(t *testing.T) manifestShape {
	t.Helper()
	var shape manifestShape
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}
	return shape
}

// The plugin runs in the editor, asks the network for the release feed, and draws one view in the
// title bar.
func TestTheManifestDeclaresTheStatusViewTheTargetAndThePermission(t *testing.T) {
	shape := readManifest(t)

	if !slices.Equal(shape.Targets, []string{"desktop.editor"}) {
		t.Errorf("targets = %v", shape.Targets)
	}
	if !slices.Equal(shape.Permissions, []string{"link:open"}) {
		t.Errorf("permissions = %v", shape.Permissions)
	}
	if !slices.Equal(shape.Services, []string{"md.numen.tools.updater"}) {
		t.Errorf("services = %v", shape.Services)
	}
	if len(shape.Contributes.Slots) != 1 || shape.Contributes.Slots[0].Slot != "titleBar.status" || shape.Contributes.Slots[0].View != "update" {
		t.Errorf("slots = %+v", shape.Contributes.Slots)
	}
}

// The settings are the ones the backend and the frontend read.
func TestTheManifestDeclaresTheSettingsThePluginReads(t *testing.T) {
	var keys []string
	for _, one := range readManifest(t).Settings {
		keys = append(keys, one.Key)
	}
	slices.Sort(keys)

	if !slices.Equal(keys, []string{"channel", "checkIntervalHours", "endpoint"}) {
		t.Errorf("settings = %v", keys)
	}
}
