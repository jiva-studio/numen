package onnxruntime

import (
	"os"
	"path/filepath"

	ort "github.com/getcharzp/onnxruntime_purego"
)

const coreMLCacheDir = "numen/coreml"

// GetCoreMLCacheDir is the folder CoreML keeps compiled models in: inside this
// platform's cache directory, readable by this user alone. It is created when
// absent.
func GetCoreMLCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, filepath.FromSlash(coreMLCacheDir))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// NewSession reads a model. A model that fails to load while compiled models are
// held in the cache directory is read again once with that directory emptied.
func NewSession(engine *ort.Engine, path string, opts *ort.SessionOptions) (*ort.Session, error) {
	session, err := engine.NewSession(path, opts)
	if err == nil {
		return session, nil
	}
	if !deleteCoreMLCache() {
		return nil, err
	}
	return engine.NewSession(path, opts)
}

// deleteCoreMLCache empties the CoreML cache directory and reports whether it
// held anything.
func deleteCoreMLCache() bool {
	dir, err := GetCoreMLCacheDir()
	if err != nil {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return false
	}
	for _, entry := range entries {
		if os.RemoveAll(filepath.Join(dir, entry.Name())) != nil {
			return false
		}
	}
	return true
}
