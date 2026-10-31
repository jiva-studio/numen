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
		StatusViews []struct {
			ID string `json:"id"`
		} `json:"statusViews"`
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
// status bar.
func TestTheManifestDeclaresTheStatusViewTheTargetAndThePermission(t *testing.T) {
	shape := readManifest(t)

	if !slices.Equal(shape.Targets, []string{"desktop.editor"}) {
		t.Errorf("targets = %v", shape.Targets)
	}
	if !slices.Equal(shape.Permissions, []string{"network:outbound"}) {
		t.Errorf("permissions = %v", shape.Permissions)
	}
	if !slices.Equal(shape.Services, []string{"numen.updater"}) {
		t.Errorf("services = %v", shape.Services)
	}
	if len(shape.Contributes.StatusViews) != 1 || shape.Contributes.StatusViews[0].ID != "update" {
		t.Errorf("status views = %+v", shape.Contributes.StatusViews)
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
