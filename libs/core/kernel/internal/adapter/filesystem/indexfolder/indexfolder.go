// Package indexfolder prepares the place the index database lives in.
package indexfolder

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNoPath is a configuration that names no index database.
var ErrNoPath = errors.New("the configuration names no index database: the application sets IndexPath")

// Prepare creates the folder of the index database at path and returns path. An empty path returns ErrNoPath.
func Prepare(path string) (string, error) {
	if path == "" {
		return "", ErrNoPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, nil
}
