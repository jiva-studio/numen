package antigravity

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

func createTempDir(pattern string) (string, func() error, error) {
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", noRelease, err
	}
	return dir, func() error {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove %s: %w", dir, err)
		}
		return nil
	}, nil
}

func isRunnable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return slices.Contains([]string{".exe", ".cmd", ".bat"}, strings.ToLower(filepath.Ext(path)))
	}
	return info.Mode()&0o111 != 0
}

func linkEntries(from, into, except string) error {
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
