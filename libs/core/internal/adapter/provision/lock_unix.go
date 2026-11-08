//go:build unix

package provision

import (
	"errors"
	"os"
	"syscall"
)

func acquireFileLock(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		closeErr := file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, errors.Join(errLockBusy, closeErr)
		}
		return nil, errors.Join(err, closeErr)
	}
	return func() error {
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
			return errors.Join(err, file.Close())
		}
		return file.Close()
	}, nil
}
