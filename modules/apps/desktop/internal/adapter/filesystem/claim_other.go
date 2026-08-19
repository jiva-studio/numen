//go:build !unix

package filesystem

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// claim takes a lock on the first byte of the file and returns what lets it go.
//
// Windows holds the lock against the open handle and drops it when the handle
// closes, which the system does for every handle a process leaves behind. The
// file is somewhere for the lock to live and says nothing on its own.
func claim(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	handle := windows.Handle(file.Fd())
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
