//go:build unix

package settings

import (
	"errors"
	"os"
	"syscall"
)

// lockFile waits for the exclusive advisory lock on the file at path, creating the file when it
// is missing, and returns the function that releases it. The kernel drops the lock when the
// process ends, however it ends.
func lockFile(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return file.Close, nil
}
