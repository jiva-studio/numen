package onnxruntime

import (
	"errors"
	"os"
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

// ErrNoHome is a library asked for where no models home is set.
var ErrNoHome = errors.New("no models folder: the host granted none and NUMEN_MODELS_DIR is not set")

// GetHome returns the models home EnvDir names, or ErrNoHome. The application passes the folder it granted through Config.Dir.
func GetHome() (string, error) {
	if dir := os.Getenv(EnvDir); dir != "" {
		return dir, nil
	}
	return "", ErrNoHome
}

// IsOffline reports whether EnvOffline forbids fetching.
func IsOffline() bool {
	return slices.Contains(getOfflineValues(), strings.ToLower(strings.TrimSpace(os.Getenv(EnvOffline))))
}

func getOfflineValues() []string { return []string{"1", "true", "yes", "on"} }

// FromEnvironment returns the configuration the environment names: the
// provider, and whether fetching is forbidden. The models home is resolved when
// the runtime is provisioned.
func FromEnvironment(clock machineport.Clock) Config {
	return Config{Provider: os.Getenv(EnvProvider), Offline: IsOffline(), Clock: clock}
}
