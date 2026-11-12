package onnxruntime

import (
	"errors"
	"os"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// The environment variables a user sets to relocate the models home, disable
// fetching or choose the execution provider. The models home and the offline
// switch are read by every plugin, so they name one folder and one rule.
const (
	// EnvDir names the folder the models are kept in.
	EnvDir = "NUMEN_MODELS_DIR"
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

// FromEnvironment returns the configuration the environment names: the
// provider, and whether fetching is forbidden. The models home is resolved when
// the runtime is provisioned.
func FromEnvironment(clock machineport.Clock) Config {
	return Config{Provider: os.Getenv(EnvProvider), Offline: provision.IsOffline(), Clock: clock}
}

// EnvMockML names the variable that runs the plugins on mock models.
const EnvMockML = "NUMEN_MOCK_ML"

// IsMockML reports whether EnvMockML runs the plugins on mock models.
func IsMockML() bool { return os.Getenv(EnvMockML) == "true" }

// GetModelsDir returns the folder EnvDir names, or defaultDir while it names none.
func GetModelsDir(defaultDir string) string {
	if models := os.Getenv(EnvDir); models != "" {
		return models
	}
	return defaultDir
}

// GetPlaceholderEnv returns the variables a mock run hands a plugin for a runtime whose placeholder is at primary:
// the placeholder as the library, on the CPU provider.
func GetPlaceholderEnv(primary string) map[string]string {
	return map[string]string{EnvLibrary: primary, EnvProvider: "cpu"}
}
