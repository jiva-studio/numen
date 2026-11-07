package keychain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// lockSuffix is added to the name of the credentials file to name the file its writers lock, which
// lies beside the credentials folder. The credentials file cannot be locked, since every write
// renames a new file over it.
const lockSuffix = ".lock"

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

// lockFile takes the exclusive lock on the file at path, creating the file when it is missing, and
// returns the function that releases it. The wait ends with the context. The system call that
// waits cannot be stopped, so it is left to finish by itself: it releases the lock as soon as it
// takes it, and the file is closed then.
func lockFile(ctx context.Context, path string) (func() error, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	locked := make(chan error, 1)
	go func() { locked <- lockOpenFile(file) }()
	select {
	case err := <-locked:
		if err != nil {
			return nil, errors.Join(err, file.Close())
		}
		return file.Close, nil
	case <-ctx.Done():
		go func() {
			<-locked
			if err := file.Close(); err != nil {
				return
			}
		}()
		return nil, ctx.Err()
	}
}
