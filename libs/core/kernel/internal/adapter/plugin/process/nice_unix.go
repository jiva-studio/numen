//go:build linux || darwin

package process

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

func markNiceDenied(err error) error {
	if errors.Is(err, unix.EACCES) || errors.Is(err, unix.EPERM) {
		return fmt.Errorf("%w: %w", ErrNiceDenied, err)
	}
	return err
}
