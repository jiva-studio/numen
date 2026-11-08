package keychain

import (
	"errors"
	"io/fs"
	"net"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/zalando/go-keyring"
)

func isKeychainAbsent(err error) bool {
	if errors.Is(err, keyring.ErrUnsupportedPlatform) {
		return true
	}
	var busErr dbus.Error
	if errors.As(err, &busErr) {
		switch busErr.Name {
		case "org.freedesktop.DBus.Error.ServiceUnknown",
			"org.freedesktop.DBus.Error.NameHasNoOwner",
			"org.freedesktop.DBus.Error.NotSupported":
			return true
		}
		return false
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) || errors.Is(err, fs.ErrNotExist) {
		return true
	}
	return strings.Contains(err.Error(), "couldn't determine address of session bus")
}
