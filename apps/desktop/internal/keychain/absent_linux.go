//go:build linux

package keychain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func isSessionBusReachable() bool {
	if address := os.Getenv("DBUS_SESSION_BUS_ADDRESS"); address != "" && !strings.HasPrefix(address, "autolaunch:") {
		return true
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	_, err := os.Stat(filepath.Join(runtimeDir, "bus"))
	return err == nil
}
