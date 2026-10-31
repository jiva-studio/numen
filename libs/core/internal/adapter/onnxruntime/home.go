package onnxruntime

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// The environment variables a user sets to relocate the models home, disable
// fetching or choose the execution provider. The models home and the offline
// switch are read by every plugin, so they name one folder and one rule.
const (
	// EnvDir names the folder the models are kept in.
	EnvDir = "NUMEN_MODELS_DIR"
	// EnvOffline, when true, forbids fetching the runtime.
	EnvOffline = "NUMEN_OFFLINE"
	// EnvProvider names the execution provider plugins open sessions on.
	EnvProvider = "NUMEN_ONNX_PROVIDER"
)

// GetHome returns the models home: EnvDir, or numen/models under the
// application data directory.
func GetHome() (string, error) {
	if dir := os.Getenv(EnvDir); dir != "" {
		return dir, nil
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(config, "numen", "models"), nil
}

// IsOffline reports whether EnvOffline forbids fetching.
func IsOffline() bool {
	return slices.Contains(offlineValues, strings.ToLower(strings.TrimSpace(os.Getenv(EnvOffline))))
}

// offlineValues are the spellings of EnvOffline that forbid fetching.
var offlineValues = []string{"1", "true", "yes", "on"}

// FromEnvironment returns the configuration the environment names: the
// provider, and whether fetching is forbidden. The models home is resolved when
// the runtime is provisioned.
func FromEnvironment(clock machineport.Clock) Config {
	return Config{Provider: os.Getenv(EnvProvider), Offline: IsOffline(), Clock: clock}
}
