//go:build unix

package provision

import (
	"errors"
	"os"
	"syscall"
)

// acquireFileLock takes an advisory lock on the file and returns what lets it go.
// The kernel drops the lock when the process ends, however it ends.
func acquireFileLock(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, errLockHeld
		}
		return nil, err
	}
	return func() error {
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
			_ = file.Close()
			return err
		}
		return file.Close()
	}, nil
}
