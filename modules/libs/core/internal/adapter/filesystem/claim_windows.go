package filesystem

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// claim takes a lock on the first byte of the file and returns what lets it go.
//
// Windows holds the lock against the open handle and drops it when the handle
// closes, which the system does for every handle a process leaves behind. The
// file is somewhere for the lock to live and says nothing on its own.
//
// The handle shares the file for deletion, so a name is taken out of the store
// with the claim on it still held.
func claim(path string) (func() error, error) {
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
		file.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
			return nil, fmt.Errorf("%s: %w", path, port.ErrClaimed)
		}
		return nil, err
	}
	return func() error {
		if err := windows.UnlockFileEx(handle, 0, 1, 0, &region); err != nil {
			file.Close()
			return err
		}
		return file.Close()
	}, nil
}
