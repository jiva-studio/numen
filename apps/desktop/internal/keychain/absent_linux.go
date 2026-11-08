//go:build linux

package keychain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// isSessionBusReachable reports whether a session bus can be found the way godbus finds it: the
// address in DBUS_SESSION_BUS_ADDRESS, else the socket <runtime dir>/bus.
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
