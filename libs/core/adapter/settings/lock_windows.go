//go:build windows

package settings

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockFile waits for the exclusive lock on the first byte of the file at path, creating the file
// when it is missing, and returns the function that releases it. Windows drops the lock when the
// handle closes.
func lockFile(path string) (func() error, error) {
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
	if err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &region); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return file.Close, nil
}
