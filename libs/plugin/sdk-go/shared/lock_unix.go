//go:build unix

package shared

import (
	"errors"
	"os"
	"syscall"
)

func lockFile(path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		// gomutants:disable-next-line reason="the error this call returns is checked below, and no test can make the call fail"
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return func() error {
		return errors.Join(syscall.Flock(int(file.Fd()), syscall.LOCK_UN), file.Close())
	}, nil
}
