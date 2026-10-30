//go:build unix

package derivedfile

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// acquireFileLock takes an advisory lock on the file and returns what lets it go.
//
// The kernel holds the lock and drops it when the process ends, however it
// ends. The file is somewhere for the lock to live and says nothing on its own.
func acquireFileLock(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		closeErr := file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, errors.Join(fmt.Errorf("%s: %w", path, vaultport.ErrClaimed), closeErr)
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
