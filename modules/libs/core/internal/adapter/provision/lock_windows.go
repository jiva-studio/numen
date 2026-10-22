//go:build windows

package provision

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// acquireFileLock takes a lock on the first byte of the file and returns what lets it go.
// Windows drops the lock when the handle closes.
func acquireFileLock(path string) (func() error, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	file := os.NewFile(uintptr(handle), path)
	var region windows.Overlapped
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK | windows.LOCKFILE_FAIL_IMMEDIATELY)
	if err := windows.LockFileEx(handle, flags, 0, 1, 0, &region); err != nil {
		_ = file.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
			return nil, errLockHeld
		}
		return nil, err
	}
	return func() error {
		if err := windows.UnlockFileEx(handle, 0, 1, 0, &region); err != nil {
			_ = file.Close()
			return err
		}
		return file.Close()
	}, nil
}
