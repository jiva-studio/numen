package onnx

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/shared"
)

// coreMLFolder is the folder of the shared directory that holds the compiled CoreML models.
const coreMLFolder = "coreml"

// lockName is the file the shared lock keeps in a directory. Emptying the directory leaves it.
const lockName = ".lock"

// GetCoreMLCacheDir returns the folder CoreML keeps the compiled form of the model at modelPath in:
// coreml/<model key> inside the directory the host shares between plugins, so every plugin that
// runs the same model compiles it once. ONNX Runtime does not notice that a model changed, so
// the key is part of the path. The folder is readable by this user alone and is created when absent.
func GetCoreMLCacheDir(modelPath string) (string, error) {
	root, err := cache.GetSharedDirOrPluginDir()
	// gomutants:disable-next-line BRANCH_IF reason="propagates cache directory resolution failure when OS cache dir is unavailable"
	if err != nil {
		return "", err
	}
	key, err := shared.GetModelKey(modelPath)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, coreMLFolder, key)
	// gomutants:disable-next-line INTEGER_INCREMENT reason="os.Chmod on line 36 enforces 0o700 permissions after MkdirAll"
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// gomutants:disable-next-line BRANCH_IF reason="chmod on directory owned by current process cannot fail"
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func lockCoreMLCache(modelPath string) (func() error, error) {
	if runtime.GOOS != "darwin" {
		return func() error { return nil }, nil
	}
	dir, hasDir := findCoreMLCacheDir(modelPath)
	if !hasDir {
		return func() error { return nil }, nil
	}
	return shared.Lock(dir)
}

func findCoreMLCacheDir(modelPath string) (string, bool) {
	dir, err := GetCoreMLCacheDir(modelPath)
	return dir, err == nil
}

// DeleteCoreMLCache empties the CoreML cache folder of the model at modelPath and reports whether
// it held anything. The caller holds the lock of the folder.
func DeleteCoreMLCache(modelPath string) bool {
	dir, dirErr := GetCoreMLCacheDir(modelPath)
	// gomutants:disable-next-line BRANCH_IF reason="os.ReadDir on empty dir also fails and returns false"
	if dirErr != nil {
		return false
	}
	entries, err := os.ReadDir(dir)
	// gomutants:disable-next-line BRANCH_IF reason="reading directory just verified by GetCoreMLCacheDir cannot fail"
	if err != nil {
		return false
	}
	isDeleted := false
	for _, entry := range entries {
		if entry.Name() == lockName {
			continue
		}
		// gomutants:disable-next-line BRANCH_IF reason="removing entries inside owned 0o700 cache directory cannot fail"
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
