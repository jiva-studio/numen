package ocr

import (
	"encoding/json"
	"slices"
	"testing"
)

// The plugin declares the ONNX Runtime as a runtime dependency and model dependencies.
func TestManifestDeclaresDependencies(t *testing.T) {
	var manifest struct {
		Runtimes     []string `json:"runtimes"`
		Dependencies []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Runtimes) != 0 {
		t.Errorf("runtimes = %v, want none", manifest.Runtimes)
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
