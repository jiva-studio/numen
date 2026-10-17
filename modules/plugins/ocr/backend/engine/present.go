package engine

import (
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
	"github.com/jiva-studio/numen/modules/plugins/ocr/backend/onnxruntime"
)

// IsFetched says whether one recogniser model file is on this machine.
func IsFetched(cfg Config, held RecogniserModel) bool {
	if held.Path != "" {
		return isPresent(held.Path)
	}
	if held.Name == "" {
		return false
	}
	for _, at := range onnxruntime.GetPaths(cfg.Dir, filepath.Base(held.Name)) {
		if isPresent(at) {
			return true
		}
	}
	if !models.IsAddress(held.Name) {
		return false
	}
	dir, err := onnxruntime.CacheDir(cfg.settings())
	if err != nil {
		return false
	}
	return isPresent(filepath.Join(dir, models.GetCacheName(held.Name, getPinnedSum(held.Name))))
}

func isPresent(at string) bool {
	_, err := os.Stat(at)
	return err == nil
}
