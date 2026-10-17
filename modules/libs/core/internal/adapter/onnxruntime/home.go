package onnxruntime

import (
	"os"
	"path/filepath"
	"strings"
)

// The environment variables a person sets to move the models home, switch
// fetching off or choose the execution provider. The models home and the offline
// switch are the ones every plugin reads, so they name one folder and one rule.
const (
	// EnvDir names the folder the models are kept in.
	EnvDir = "NUMEN_MODELS_DIR"
	// EnvOffline, when true, forbids fetching the runtime.
	EnvOffline = "NUMEN_OFFLINE"
	// EnvProvider names the execution provider plugins open sessions on.
	EnvProvider = "NUMEN_ONNX_PROVIDER"
)

// homeDir is the models folder under the application data directory.
const homeDir = "numen/models"

// GetHome returns the models home: EnvDir, or numen/models under the
// application data directory.
func GetHome() (string, error) {
	if dir := os.Getenv(EnvDir); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, filepath.FromSlash(homeDir)), nil
}

// IsOffline reports whether EnvOffline forbids fetching.
func IsOffline() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvOffline))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// FromEnvironment returns the configuration the environment names: the
// provider, and whether fetching is forbidden. The models home is resolved when
// the runtime is provisioned.
func FromEnvironment() Config {
	return Config{Provider: os.Getenv(EnvProvider), Offline: IsOffline()}
}
