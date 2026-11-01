package embeddings

import (
	"encoding/json"
	"slices"
	"testing"
)

// The plugin declares the ONNX Runtime as a runtime dependency and no legacy runtimes list.
func TestManifestDeclaresONNXRuntimeDependency(t *testing.T) {
	var manifest struct {
		Runtimes     []string `json:"runtimes"`
		Dependencies []struct {
			ID        string `json:"id"`
			Kind      string `json:"kind"`
			SourceURL string `json:"sourceUrl"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Runtimes) != 0 {
		t.Errorf("runtimes = %v, want none", manifest.Runtimes)
	}
	if len(manifest.Dependencies) != 1 || manifest.Dependencies[0].ID != "onnxruntime" || manifest.Dependencies[0].Kind != "runtime" || manifest.Dependencies[0].SourceURL != "" {
		t.Errorf("dependencies = %+v, want one onnxruntime runtime", manifest.Dependencies)
	}
}

// The core finds the provider by its capability; no window calls the service.
func TestManifestDeclaresTheEmbeddingCapabilityAndNoWindowService(t *testing.T) {
	var manifest struct {
		Capabilities []string `json:"capabilities"`
		Services     []string `json:"services"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(manifest.Capabilities, "embedding") {
		t.Errorf("capabilities = %v, want embedding", manifest.Capabilities)
	}
	if len(manifest.Services) != 0 {
		t.Errorf("serves %v to the windows, want nothing", manifest.Services)
	}
}

// The plugin reports the model and a starting floor for it and draws nothing: the search floor is
// measured and set by another plugin.
func TestManifestDeclaresNoFrontendAndNoSettingsView(t *testing.T) {
	var manifest struct {
		Frontend      map[string]any   `json:"frontend"`
		SettingsViews []map[string]any `json:"settingsViews"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Frontend) != 0 || len(manifest.SettingsViews) != 0 {
		t.Errorf("frontend %v and settingsViews %v, want neither", manifest.Frontend, manifest.SettingsViews)
	}
}
