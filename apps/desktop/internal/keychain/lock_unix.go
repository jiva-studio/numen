//go:build unix

package keychain

import (
	"context"
	"errors"
	"os"
	"syscall"
)

// lockFile waits for the exclusive advisory lock on the file at path, creating the file when it
// is missing, and returns the function that releases it. The wait ends with the context. The
// kernel drops the lock when the process ends, however it ends.
func lockFile(ctx context.Context, path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	err = waitForLock(ctx, func() (bool, error) {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EINTR) {
			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return file.Close, nil
}
