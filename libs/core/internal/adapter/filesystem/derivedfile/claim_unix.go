//go:build unix

package derivedfile

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// acquireFileLock takes an advisory lock on the file and returns the function
// that releases it.
//
// The kernel holds the lock and releases it when the process ends, however it
// ends. The file only anchors the lock and its content is not used.
func acquireFileLock(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%s: %w", path, vaultport.ErrClaimed)
		}
		return nil, err
	}
	return func() error {
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
			file.Close()
			return err
		}
		return file.Close()
	}, nil
}
