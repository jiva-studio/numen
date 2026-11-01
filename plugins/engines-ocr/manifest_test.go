package ocr

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

// The plugin declares the ONNX Runtime as a runtime dependency and model dependencies.
func TestManifestDeclaresDependencies(t *testing.T) {
	manifest := plugintest.ReadManifest(t, Manifest)
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
