package embeddings

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

// The plugin declares the ONNX Runtime as its one runtime dependency.
func TestManifestDeclaresONNXRuntimeDependency(t *testing.T) {
	manifest := plugintest.ReadManifest(t, Manifest)
	if len(manifest.Dependencies) != 1 || manifest.Dependencies[0].ID != "onnxruntime" || manifest.Dependencies[0].Kind != "runtime" || manifest.Dependencies[0].SourceURL != "" {
		t.Errorf("dependencies = %+v, want one onnxruntime runtime", manifest.Dependencies)
	}
}

// The core finds the provider by its capability; no window calls the service.
func TestManifestDeclaresTheEmbeddingCapabilityAndNoWindowService(t *testing.T) {
	manifest := plugintest.ReadManifest(t, Manifest)
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
	manifest := plugintest.ReadManifest(t, Manifest)
	if manifest.Frontend != nil || len(manifest.SettingsViews) != 0 {
		t.Errorf("frontend %v and settingsViews %v, want neither", manifest.Frontend, manifest.SettingsViews)
	}
}
