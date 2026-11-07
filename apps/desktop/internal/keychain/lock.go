package keychain

import (
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
// process and in any other, so that a write reads the file after the writes before it ended.
func (s *Store) changeEntries(work func() error) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("lock the credentials file: %w", err)
	}
	release, err := lockFile(filepath.Join(s.dir, credentialsFile+lockSuffix))
	if err != nil {
		return fmt.Errorf("lock the credentials file: %w", err)
	}
	defer func() { err = errors.Join(err, release()) }()
	return work()
}
