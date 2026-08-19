//go:build unix

package filesystem

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// claim takes an advisory lock on the file and returns what lets it go.
//
// The kernel holds the lock and drops it when the process ends, however it
// ends. The file is somewhere for the lock to live and says nothing on its own.
func claim(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%s: %w", path, port.ErrClaimed)
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
