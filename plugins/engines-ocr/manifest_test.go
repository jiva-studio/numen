package ocr

import (
	"encoding/json"
	"slices"
	"testing"
)

// The plugin declares the ONNX Runtime as a runtime dependency and model dependencies.
func TestManifestDeclaresDependencies(t *testing.T) {
	var manifest struct {
		Dependencies []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, dep := range manifest.Dependencies {
		ids = append(ids, dep.ID)
	}
	for _, expected := range []string{"onnxruntime", "pp-doclayout", "pp-ocr-det", "rec-pp-ocrv6-tiny", "rec-pp-ocrv5-eslav"} {
		if !slices.Contains(ids, expected) {
			t.Errorf("dependencies missing %s: got %v", expected, ids)
		}
	}
}

// The process keeps its loaded model for ten minutes without a call.
func TestTheManifestKeepsTheProcessForTenIdleMinutes(t *testing.T) {
	var manifest struct {
		Backend struct {
			IdleStopSeconds int `json:"idleStopSeconds"`
		} `json:"backend"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}

	if manifest.Backend.IdleStopSeconds != 600 {
		t.Errorf("idleStopSeconds = %d, want 600", manifest.Backend.IdleStopSeconds)
	}
}

// Recognize automatically wakes the plugin on a new or changed scan; the scans to recognize are read
// from the file change feed, so no trigger recognizes the file it runs on.
func TestTheTriggersWakeTheCommandThatReadsTheFeed(t *testing.T) {
	var manifest struct {
		Contributes struct {
			Triggers []struct {
				Event   string `json:"event"`
				Command string `json:"command"`
			} `json:"triggers"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	var events []string
	for _, one := range manifest.Contributes.Triggers {
		if one.Command != "recognitions.update" {
			t.Errorf("a trigger on %s runs %s", one.Event, one.Command)
		}
		events = append(events, one.Event)
	}
	if !slices.Equal(events, []string{"file.added", "file.changed"}) {
		t.Errorf("the triggers run on %v, want file.added and file.changed", events)
	}
}
