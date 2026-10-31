package transcription

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

// The plugin declares the ONNX Runtime and model dependencies with checksums.
func TestManifestDeclaresDependencies(t *testing.T) {
	plugintest.CheckDependencies(t, Manifest, true,
		"onnxruntime", "silero-vad",
		"parakeet-v3-encoder", "parakeet-v3-decoder", "parakeet-v3-joiner", "parakeet-v3-tokens",
		"parakeet-v2-encoder", "parakeet-v2-decoder", "parakeet-v2-joiner", "parakeet-v2-tokens")
}
