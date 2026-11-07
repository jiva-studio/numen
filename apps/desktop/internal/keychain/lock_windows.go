//go:build windows

package keychain

import (
	"context"
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockFile waits for the exclusive lock on the first byte of the file at path, creating the file
// when it is missing, and returns the function that releases it. The wait ends with the context.
// Windows drops the lock when the handle closes.
func lockFile(ctx context.Context, path string) (func() error, error) {
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
	err = waitForLock(ctx, func() (bool, error) {
		var region windows.Overlapped
		err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &region)
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return file.Close, nil
}
