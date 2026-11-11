// Package process reads the environment of the plugin process and writes its standard error.
package process

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
)

// EnvMockML names the variable that has the plugin read every page with the mock recogniser.
const EnvMockML = "NUMEN_MOCK_ML"

// Process is the process the plugin runs in.
type Process struct{}

// IsMockML reports whether the environment asks for the mock recogniser.
func (Process) IsMockML() bool {
	return os.Getenv(EnvMockML) == "true"
}

// GetConfig returns the default configuration with the folder and the runtime the environment names.
func (Process) GetConfig() engine.Config {
	cfg := engine.Defaults()
	if dir := os.Getenv("NUMEN_OCR_DIR"); dir != "" {
		cfg.Dir = dir
	} else if modelsDir := os.Getenv(engine.EnvDir); modelsDir != "" {
		cfg.Dir = modelsDir
	}
	if runtime := os.Getenv("NUMEN_OCR_RUNTIME"); runtime != "" {
		cfg.Runtime = runtime
	}
	return cfg
}

// ReportCloseError writes the failure to close the recogniser to standard error.
func (Process) ReportCloseError(err error) {
	fmt.Fprintln(os.Stderr, "engines-ocr: close the recogniser:", err)
}
