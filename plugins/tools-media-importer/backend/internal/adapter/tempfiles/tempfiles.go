// Package tempfiles manages temporary files and directories for media downloads.
package tempfiles

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Files reads and writes temporary files and directories on the machine.
type Files struct{}

// WriteTemp writes content to a new temporary file matching pattern and returns its path.
func (Files) WriteTemp(pattern string, content []byte) (string, error) {
	tempFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	_, writeErr := tempFile.Write(content)
	if err := errors.Join(writeErr, tempFile.Close()); err != nil {
		return "", errors.Join(err, os.Remove(tempFile.Name()))
	}
	return tempFile.Name(), nil
}

// ReadFile reads the named file and returns the contents.
func (Files) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// ReadDir reads the named directory and returns the names of its entries.
func (Files) ReadDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}

// CopyFile copies the contents of path into into.
func (Files) CopyFile(path string, into io.Writer) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	_, err = io.Copy(into, file)
	return err
}

// WithTempDir creates a temporary directory matching pattern, calls fn with its path, and removes it.
func (Files) WithTempDir(pattern string, fn func(dir string) error) (err error) {
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	return fn(dir)
}

// Join joins the path of a directory and the name of a file in it with the separator of the machine.
func (Files) Join(dir, name string) string {
	return filepath.Join(dir, name)
}
