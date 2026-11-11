// Package process reads the environment of the plugin process.
package process

import (
	"os"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/settings"
)

// EnvMockML names the variable that has the plugin hear every recording with the mock model.
const EnvMockML = "NUMEN_MOCK_ML"

// envModelsDir names the folder plugins keep their models in; the host sets it.
const envModelsDir = "NUMEN_MODELS_DIR"

// envONNXRuntime names the variable that holds the path of the ONNX runtime library.
const envONNXRuntime = "NUMEN_ONNX_RUNTIME"

// IsMockML reports whether the variable EnvMockML in vars asks for the mock model.
func IsMockML(vars map[string]string) bool {
	return vars[EnvMockML] == "true"
}

// Process is the process the plugin runs in.
type Process struct{}

// IsMockML reports whether the environment asks for the mock model.
func (Process) IsMockML() bool {
	return os.Getenv(EnvMockML) == "true"
}

// GetConfig returns the default configuration with the folder and the runtime the environment names.
func (Process) GetConfig() settings.Config {
	cfg := settings.Defaults()
	if dir := os.Getenv(envModelsDir); dir != "" {
		cfg.Dir = dir
	}
	if runtime := os.Getenv(envONNXRuntime); runtime != "" {
		cfg.Runtime = runtime
	}
	return cfg
}
