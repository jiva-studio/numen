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
