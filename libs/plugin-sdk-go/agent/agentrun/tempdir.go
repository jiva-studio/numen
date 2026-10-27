package agentrun

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// CreateTempDir creates a temporary directory matching pattern, and returns its path and a clean-up
// function that deletes it.
func CreateTempDir(pattern string) (string, func(), error) {
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", func() {}, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

// WriteTempFile creates a mode-0600 temporary file matching pattern, writes data to it, and returns
// its path.
func WriteTempFile(pattern string, data []byte) (string, error) {
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("create temp config: %w", err)
	}
	defer file.Close()

	if err := file.Chmod(0o600); err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("chmod config: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("write config: %w", err)
	}
	return file.Name(), nil
}

// RemoveFile removes the file at path when present.
func RemoveFile(path string) {
	// gomutants:disable-next-line BRANCH_IF reason="os.Remove on an empty path returns ErrNotExist which is ignored"
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "remove %s: %v\n", path, err)
	}
}

// IsRunnable reports whether path is an executable file.
func IsRunnable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		return slices.Contains([]string{".exe", ".cmd", ".bat"}, ext)
	}
	return info.Mode()&0o111 != 0
}

// ReadFile reads the file at path.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to the file at path with permissions perm.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// CreateDir creates the directory at path along with any parent directories.
func CreateDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// LinkEntries symlinks each entry in from into into, skipping except.
func LinkEntries(from, into, except string) {
	entries, err := os.ReadDir(from)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if name != except {
			_ = os.Symlink(filepath.Join(from, name), filepath.Join(into, name))
		}
	}
}
