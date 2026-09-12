package recognition

import (
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// IsFetched says whether one recogniser's file is on this machine: the path
// written down, a file beside the application, or what an address is kept under
// in the download cache.
//
// It stats what a fetch would have written, and opens, reaches and creates
// nothing.
func IsFetched(cfg Config, held RecogniserModel) bool {
	if held.Path != "" {
		return stands(held.Path)
	}
	if held.Name == "" {
		return false
	}
	for _, at := range onnxruntime.Beside(cfg.Dir, filepath.Base(held.Name)) {
		if stands(at) {
			return true
		}
	}
	if !onnxruntime.IsAddress(held.Name) {
		return false
	}
	dir, err := onnxruntime.CacheDir(cfg.settings())
	if err != nil {
		return false
	}
	return stands(filepath.Join(dir, onnxruntime.GetCacheName(held.Name)))
}

func stands(at string) bool {
	_, err := os.Stat(at)
	return err == nil
}
