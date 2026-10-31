package onnx

import (
	"os"
	"path/filepath"
)

const coreMLCacheDir = "numen/coreml"

// GetCoreMLCacheDir returns the folder CoreML keeps compiled models in: inside
// this platform's cache directory, readable by this user alone. It is created
// when absent.
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

// DeleteCoreMLCache empties the CoreML cache directory and reports whether it
// held anything.
func DeleteCoreMLCache() bool {
	dir, dirErr := GetCoreMLCacheDir()
	if dirErr != nil {
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
