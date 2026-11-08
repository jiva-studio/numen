package keychain

import (
	"errors"

	"github.com/godbus/dbus/v5"
	"github.com/zalando/go-keyring"
)

func isKeychainAbsent(err error) bool {
	if errors.Is(err, keyring.ErrUnsupportedPlatform) {
		return true
	}
	if busErr, ok := errors.AsType[dbus.Error](err); ok {
		switch busErr.Name {
		case "org.freedesktop.DBus.Error.ServiceUnknown",
			"org.freedesktop.DBus.Error.NameHasNoOwner",
			"org.freedesktop.DBus.Error.NotSupported":
			return true
		}
		return false
	}
	return !isSessionBusReachable()
}
