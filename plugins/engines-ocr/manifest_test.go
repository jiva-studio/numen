package ocr

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

// The plugin declares the ONNX Runtime as a runtime dependency and model dependencies.
func TestManifestDeclaresDependencies(t *testing.T) {
	plugintest.CheckDependencies(t, Manifest, false,
		"onnxruntime", "pp-doclayout", "pp-ocr-det", "rec-pp-ocrv6-tiny", "rec-pp-ocrv5-eslav")
}
