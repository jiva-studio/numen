package keychain

import (
	"errors"
	"io/fs"
	"net"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/zalando/go-keyring"
)

// dbusMissing are the errors of the message bus that say no secret service is there to answer.
var dbusMissing = []string{
	"org.freedesktop.DBus.Error.ServiceUnknown",
	"org.freedesktop.DBus.Error.NameHasNoOwner",
	"org.freedesktop.DBus.Error.NotSupported",
}

func isKeychainAbsent(err error) bool {
	if errors.Is(err, keyring.ErrUnsupportedPlatform) {
		return true
	}
	var busErr dbus.Error
	if errors.As(err, &busErr) {
		for _, name := range dbusMissing {
			if busErr.Name == name {
				return true
			}
		}
		return false
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) || errors.Is(err, fs.ErrNotExist) {
		return true
	}
	return strings.Contains(err.Error(), "couldn't determine address of session bus")
}
