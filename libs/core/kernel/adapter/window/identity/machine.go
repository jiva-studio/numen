package identity

import (
	"os"
	"runtime"
	"strings"
)

// hostName names this machine by its host name, or by its OS and architecture where it has none.
type hostName struct{}

func (hostName) GetInstanceName() string {
	if host, err := os.Hostname(); err == nil {
		if trimmed := strings.TrimSpace(host); trimmed != "" {
			return trimmed
		}
	}
	return runtime.GOOS + "-" + runtime.GOARCH
}
