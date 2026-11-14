//go:build unix

package shared

import (
	"errors"
	"os"
	"syscall"
)

func lockFile(path string) (func() error, error) {
	return lockFileWith(path, syscall.Flock)
}

func lockFileWith(path string, flock func(fd, how int) error) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		err = flock(int(file.Fd()), syscall.LOCK_EX)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return func() error {
		return errors.Join(flock(int(file.Fd()), syscall.LOCK_UN), file.Close())
	}, nil
}
