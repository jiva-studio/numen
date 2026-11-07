//go:build windows

package keychain

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockOpenFile waits for the exclusive lock on the first byte of the file. Windows drops the lock
// when the handle closes.
func lockOpenFile(file *os.File) error {
	var region windows.Overlapped
	return windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &region)
}
