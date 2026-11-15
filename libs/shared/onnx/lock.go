package onnx

import (
	"fmt"
	"os"
	"path/filepath"
)

func lockDir(dir string) (release func() error, err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	release, err = lockFile(filepath.Join(dir, lockName))
	if err != nil {
		return nil, fmt.Errorf("lock %s: %w", dir, err)
	}
	return release, nil
}
