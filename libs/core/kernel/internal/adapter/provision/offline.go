package provision

import (
	"os"
	"strings"
)

// EnvOffline, when true, forbids fetching a dependency.
const EnvOffline = "NUMEN_OFFLINE"

// IsOffline reports whether EnvOffline forbids fetching.
func IsOffline() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvOffline))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// EnvMockML names the variable that runs the plugins on mock models.
const EnvMockML = "NUMEN_TEST_MOCK_ML"

// IsMockML reports whether EnvMockML runs the plugins on mock models.
func IsMockML() bool { return os.Getenv(EnvMockML) == "true" }

// EnvModelsDir names the folder the user moves the models and the downloaded dependencies to.
const EnvModelsDir = "NUMEN_MODELS_DIR"

// GetModelsDir returns the folder EnvModelsDir names, or defaultDir while it names none.
func GetModelsDir(defaultDir string) string {
	if models := os.Getenv(EnvModelsDir); models != "" {
		return models
	}
	return defaultDir
}
