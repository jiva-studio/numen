package ocr

import (
	"encoding/json"
	"slices"
	"strings"
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

// The onFilesChanged event wakes the plugin, which reads the files to work from the file change feed;
// the plugin has no trigger.
func TestTheFilesChangedEventWakesThePlugin(t *testing.T) {
	var manifest struct {
		ActivationEvents []string        `json:"activationEvents"`
		Contributes      json.RawMessage `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(manifest.ActivationEvents, []string{"onFilesChanged:application/pdf", "onFilesChanged:image/png", "onFilesChanged:image/jpeg", "onFilesChanged:image/gif", "onFilesChanged:image/webp"}) {
		t.Errorf("activationEvents = %v, want the masks of pdf and the image types", manifest.ActivationEvents)
	}
	if strings.Contains(string(manifest.Contributes), "triggers") {
		t.Errorf("contributes = %s, want no trigger", manifest.Contributes)
	}
}
