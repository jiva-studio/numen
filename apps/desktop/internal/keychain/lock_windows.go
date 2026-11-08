//go:build windows

package keychain

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func tryLockFile(file *os.File) error {
	var region windows.Overlapped
	err := windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &region)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return errLocked
	}
	return err
}
