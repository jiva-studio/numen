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
// function that deletes it and returns the error of the deletion.
func CreateTempDir(pattern string) (string, func() error, error) {
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", func() error { return nil }, err
	}
	return dir, func() error {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove %s: %w", dir, err)
		}
		return nil
	}, nil
}

// WriteTempFile creates a mode-0600 temporary file matching pattern, writes data to it, and returns
// its path.
func WriteTempFile(pattern string, data []byte) (path string, err error) {
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("create temp config: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close config: %w", closeErr))
		}
		if err != nil {
			if removeErr := os.Remove(file.Name()); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove config: %w", removeErr))
			}
			path = ""
		}
	}()

	if err := file.Chmod(0o600); err != nil {
		return "", fmt.Errorf("chmod config: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	return file.Name(), nil
}

// RemoveFile removes the file at path. An empty path and a missing file are not errors.
func RemoveFile(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
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

// LinkEntries symlinks each entry in from into into, skipping except. A missing from folder and an
// entry already present in into are not errors. A failing entry does not stop the remaining ones;
// the failures are returned joined.
func LinkEntries(from, into, except string) error {
	entries, err := os.ReadDir(from)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %s: %w", from, err)
	}
	var failures []error
	for _, entry := range entries {
		name := entry.Name()
		if name == except {
			continue
		}
		target := filepath.Join(into, name)
		if err := os.Symlink(filepath.Join(from, name), target); err != nil && !errors.Is(err, fs.ErrExist) {
			failures = append(failures, fmt.Errorf("link %s: %w", target, err))
		}
	}
	return errors.Join(failures...)
}
