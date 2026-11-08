//go:build unix

package keychain

import (
	"errors"
	"os"
	"syscall"
)

func lockOpenFile(file *os.File) error {
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
		if !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}
