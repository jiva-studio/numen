// Package shared holds what plugins that use the directory the host shares between all plugins
// agree on: the lock one writer of an entry takes, and the key a model is stored under.
package shared

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// lockName is the file inside a directory that holds its lock.
const lockName = ".lock"

// Lock takes the exclusive lock of the directory dir, which it creates if it is missing, and waits
// for a holder in this or another process to release it. The lock is an operating-system file
// lock (flock on Unix, LockFileEx on Windows), so a process that ends releases it. release lets it
// go and closes the lock file.
func Lock(dir string) (release func() error, err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	release, err = lockFile(filepath.Join(dir, lockName))
	if err != nil {
		return nil, fmt.Errorf("lock %s: %w", dir, err)
	}
	return release, nil
}

// readyName is the file the host writes beside a model it provisioned. Its first line names the
// model file and its second the SHA-256 the model was verified against.
const readyName = ".ready"

// GetModelKey returns the name the data derived from a model file is stored under. It is the SHA-256 the host
// verified the file against when the host provisioned the file; the file is never read. For a file
// the host did not provision, it is a digest of the file's name, size and modification time, which
// changes when the file does.
func GetModelKey(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if sum, ok := readProvisionedSum(path); ok {
		return sum, nil
	}
	sum := sha256.Sum256(fmt.Appendf(nil, "%s\n%d\n%d", filepath.Base(path), info.Size(), info.ModTime().UnixNano()))
	return "stat-" + hex.EncodeToString(sum[:]), nil
}

func readProvisionedSum(path string) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), readyName))
	// gomutants:disable-next-line BRANCH_IF reason="ReadFile returns no data with its error, and no data has fewer than two lines, so the check below returns the same empty sum and false"
	if err != nil {
		return "", false
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != filepath.Base(path) || !isDigest(strings.TrimSpace(lines[1])) {
		return "", false
	}
	return strings.ToLower(strings.TrimSpace(lines[1])), true
}

func isDigest(text string) bool {
	if len(text) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(text)
	return err == nil
}
