//go:build unix

package keychain

import (
	"errors"
	"os"
	"syscall"
)

// lockOpenFile waits for the exclusive advisory lock on the file. The kernel drops the lock when
// the file closes or the process ends, however it ends.
func lockOpenFile(file *os.File) error {
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
		if !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}
