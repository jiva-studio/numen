//go:build windows

package shared

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32     = syscall.NewLazyDLL("kernel32.dll")
	lockFileEx   = kernel32.NewProc("LockFileEx")
	unlockFileEx = kernel32.NewProc("UnlockFileEx")
)

func lockFile(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	handle := syscall.Handle(file.Fd())
	var region syscall.Overlapped
	const exclusive = 0x2
	if result, _, callErr := lockFileEx.Call(uintptr(handle), exclusive, 0, 1, 0, uintptr(unsafe.Pointer(&region))); result == 0 {
		return nil, errors.Join(callErr, file.Close())
	}
	return func() error {
		var unlocked error
		if result, _, callErr := unlockFileEx.Call(uintptr(handle), 0, 1, 0, uintptr(unsafe.Pointer(&region))); result == 0 {
			unlocked = callErr
		}
		return errors.Join(unlocked, file.Close())
	}, nil
}
