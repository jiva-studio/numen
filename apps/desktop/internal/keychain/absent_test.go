package keychain

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"runtime"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/zalando/go-keyring"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func useReachableSessionBus(t *testing.T) {
	t.Helper()
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/numen-test-bus")
}

func useNoSessionBus(t *testing.T) {
	t.Helper()
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
}

func TestTheErrorsOfAMissingKeychainAreAbsent(t *testing.T) {
	useReachableSessionBus(t)
	for name, err := range map[string]error{
		"unsupported platform": fmt.Errorf("no keychain: %w", keyring.ErrUnsupportedPlatform),
		"service unknown":      dbus.Error{Name: "org.freedesktop.DBus.Error.ServiceUnknown"},
		"name has no owner":    fmt.Errorf("get: %w", dbus.Error{Name: "org.freedesktop.DBus.Error.NameHasNoOwner"}),
		"not supported":        dbus.Error{Name: "org.freedesktop.DBus.Error.NotSupported"},
	} {
		if !isKeychainAbsent(err) {
			t.Errorf("%s: the keychain is not reported absent", name)
		}
	}
}

func TestAKeychainThatIsThereAndFailsIsNotAbsent(t *testing.T) {
	useReachableSessionBus(t)
	for name, err := range map[string]error{
		"locked":            dbus.Error{Name: "org.freedesktop.Secret.Error.IsLocked"},
		"access denied":     dbus.Error{Name: "org.freedesktop.DBus.Error.AccessDenied"},
		"no reply":          dbus.Error{Name: "org.freedesktop.DBus.Error.NoReply"},
		"connection reset":  &net.OpError{Op: "read", Net: "unix", Err: errors.New("connection reset by peer")},
		"timeout":           &net.OpError{Op: "dial", Net: "unix", Err: os.ErrDeadlineExceeded},
		"socket vanished":   fmt.Errorf("dial: %w", fs.ErrNotExist),
		"bus address text":  errors.New("dbus: couldn't determine address of session bus"),
		"unrecognized text": errors.New("something else"),
	} {
		if isKeychainAbsent(err) {
			t.Errorf("%s: a keychain that is there is reported absent", name)
		}
	}
}

func TestAKeychainIsAbsentWhenNoSessionBusExists(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the session bus is looked for on Linux only")
	}
	useNoSessionBus(t)
	if !isKeychainAbsent(errors.New("dbus: couldn't determine address of session bus")) {
		t.Error("no session bus, and the keychain is not reported absent")
	}
	if !isKeychainAbsent(&net.OpError{Op: "dial", Net: "unix", Err: fs.ErrNotExist}) {
		t.Error("no session bus and a dial error, and the keychain is not reported absent")
	}
}

func TestAKeychainThatFailsIsNotReplacedByTheFile(t *testing.T) {
	useReachableSessionBus(t)
	for name, failure := range map[string]error{
		"dial error":       &net.OpError{Op: "dial", Net: "unix", Err: fs.ErrNotExist},
		"connection reset": &net.OpError{Op: "read", Net: "unix", Err: errors.New("connection reset by peer")},
		"bus address text": errors.New("dbus: couldn't determine address of session bus"),
		"locked":           dbus.Error{Name: "org.freedesktop.Secret.Error.IsLocked"},
	} {
		keyring.MockInitWithError(failure)
		dir := t.TempDir()
		store := newTestStore(dir)
		if err := store.SetKey(t.Context(), entryName, valueOne); !errors.Is(err, plugin.ErrNoKeyStore) {
			t.Errorf("%s: SetKey returned %v, want ErrNoKeyStore", name, err)
		}
		if _, err := store.GetKey(t.Context(), entryName); !errors.Is(err, plugin.ErrNoKeyStore) {
			t.Errorf("%s: GetKey returned %v, want ErrNoKeyStore", name, err)
		}
		if _, err := os.Stat(credentialsPath(dir)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: a key was written to the file: %v", name, err)
		}
	}
}
