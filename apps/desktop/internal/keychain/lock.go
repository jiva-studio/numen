package keychain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
)

// lockSuffix is added to the name of the credentials file to name the file its writers lock, which
// lies beside the credentials folder. The credentials file cannot be locked, since every write
// renames a new file over it.
const lockSuffix = ".lock"

// lockPollInterval is the time between two attempts to take the lock of another handle. The system
// call that waits for a file lock cannot be canceled, so the lock is polled.
const lockPollInterval = 20 * time.Millisecond

// errLocked is returned by tryLockFile while another handle has the lock.
var errLocked = errors.New("the file is locked")

func (s *Store) changeEntries(ctx context.Context, work func() error) (err error) {
	select {
	case s.writer <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.writer }()
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("lock the credentials file: %w", err)
	}
	file, err := lockFile(ctx, s.wait, filepath.Join(s.dir, credentialsFile+lockSuffix))
	if err != nil {
		return fmt.Errorf("lock the credentials file: %w", err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return work()
}

func lockFile(ctx context.Context, wait machine.Wait, path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		err := tryLockFile(file)
		if err == nil {
			return file, nil
		}
		if !errors.Is(err, errLocked) {
			return nil, errors.Join(err, file.Close())
		}
		if err := wait(ctx, lockPollInterval); err != nil {
			return nil, errors.Join(err, file.Close())
		}
	}
}
