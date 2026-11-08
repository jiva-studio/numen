//go:build unix

package keychain

import (
	"errors"
	"os"
	"syscall"
)

func tryLockFile(file *os.File) error {
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, syscall.EINTR):
		case errors.Is(err, syscall.EWOULDBLOCK):
			return errLocked
		default:
			return err
		}
	}
}
