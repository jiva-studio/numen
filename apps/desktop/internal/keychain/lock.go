package keychain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// lockSuffix is added to the name of the credentials file to name the file its writers lock, which
// lies beside the credentials folder. The credentials file cannot be locked, since every write
// renames a new file over it.
const lockSuffix = ".lock"

// lockPollInterval is how long a writer waits between two attempts to take the lock.
const lockPollInterval = 10 * time.Millisecond

// changeEntries runs work under an exclusive lock that every writer of the file takes, in this
// process and in any other, so that a write reads the file after the writes before it ended. The
// wait for the lock ends with the context, and the store is not locked while it lasts.
func (s *Store) changeEntries(ctx context.Context, work func() error) (err error) {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("lock the credentials file: %w", err)
	}
	release, err := lockFile(ctx, filepath.Join(s.dir, credentialsFile+lockSuffix))
	if err != nil {
		return fmt.Errorf("lock the credentials file: %w", err)
	}
	defer func() { err = errors.Join(err, release()) }()
	s.mu.Lock()
	defer s.mu.Unlock()
	return work()
}

// waitForLock calls tryLock until it takes the lock or fails, and stops with the context.
func waitForLock(ctx context.Context, tryLock func() (isLocked bool, err error)) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		isLocked, err := tryLock()
		if err != nil || isLocked {
			return err
		}
		timer.Reset(lockPollInterval)
	}
}
