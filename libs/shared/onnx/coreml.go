package onnx

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// coreMLFolder is the folder of the cache root that holds the compiled CoreML models.
const coreMLFolder = "coreml"

// lockName is the file inside a cache folder that holds its lock. Emptying the folder leaves it.
const lockName = ".lock"

// GetCoreMLCacheDir returns the folder CoreML keeps the compiled form of the model at modelPath in:
// coreml/<model key> inside cacheRoot. A caller that passes the directory the host shares between
// plugins lets every plugin that runs the same model compile it once. ONNX Runtime does not notice
// that a model changed, so the key is part of the path. The folder is readable by this user alone
// and is created when absent.
func GetCoreMLCacheDir(cacheRoot, modelPath string) (string, error) {
	if cacheRoot == "" {
		return "", errors.New("coreml cache: no cache root")
	}
	key, err := getModelKey(modelPath)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheRoot, coreMLFolder, key)
	// gomutants:disable-next-line INTEGER_INCREMENT reason="os.Chmod sets the final 0700 permissions on the directory immediately after creation."
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// gomutants:disable-next-line BRANCH_IF reason="os.Chmod on a directory just created by os.MkdirAll only fails on a concurrent filesystem change."
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func lockCoreMLCache(cacheRoot, modelPath string) (func() error, error) {
	if runtime.GOOS != "darwin" {
		return func() error { return nil }, nil
	}
	dir, hasDir := findCoreMLCacheDir(cacheRoot, modelPath)
	if !hasDir {
		return func() error { return nil }, nil
	}
	return lockDir(dir)
}

func findCoreMLCacheDir(cacheRoot, modelPath string) (string, bool) {
	dir, err := GetCoreMLCacheDir(cacheRoot, modelPath)
	return dir, err == nil
}

// DeleteCoreMLCache empties the CoreML cache folder of the model at modelPath inside cacheRoot and
// reports whether it held anything. The caller holds the lock of the folder.
func DeleteCoreMLCache(cacheRoot, modelPath string) bool {
	dir, dirErr := GetCoreMLCacheDir(cacheRoot, modelPath)
	if dirErr != nil {
		return false
	}
	entries, err := os.ReadDir(dir)
	// gomutants:disable-next-line BRANCH_IF reason="When os.ReadDir fails, entries is empty and the loop below returns false."
	if err != nil {
		return false
	}
	isDeleted := false
	for _, entry := range entries {
		if entry.Name() == lockName {
			continue
		}
		if os.RemoveAll(filepath.Join(dir, entry.Name())) != nil {
			return false
		}
		isDeleted = true
	}
	return isDeleted
}

// GetCoreMLOptions returns the option list the keyed CoreML entry point is given.
func GetCoreMLOptions(cacheDir string, units ComputeUnits) [][2]string {
	if units == ComputeUnitsDefault {
		units = ComputeUnitsAll
	}
	return [][2]string{
		{"ModelCacheDirectory", cacheDir},
		{"EnableOnSubgraphs", "1"},
		{"CreateMLProgram", "1"},
		{"MLComputeUnits", string(units)},
	}
}
