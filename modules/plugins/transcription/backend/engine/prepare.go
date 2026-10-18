package engine

import (
	"context"
	"os"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

// Ready says whether everything a transcription needs is already on this
// machine. It opens nothing and fetches nothing, so it is answered while a
// transcription is fetching what it needs.
func Ready(cfg Config) bool {
	cfg.ShouldDownload = false
	if !onnx.IsAvailable(cfg.getRuntimeLibrary()) {
		return false
	}
	var resolved paths
	for _, one := range getWantedFiles(cfg, &resolved) {
		if _, err := models.Locate(context.Background(), cfg.source(), pinnedSums, one.path, one.name, one.kind); err != nil {
			return false
		}
	}
	return true
}

// GetModelBytes is the size of the model files on this machine, and zero where
// any of them is not here. It opens nothing and fetches nothing.
func GetModelBytes(cfg Config) int64 {
	cfg.ShouldDownload = false
	var resolved paths
	var total int64
	for _, one := range getWantedFiles(cfg, &resolved) {
		at, err := models.Locate(context.Background(), cfg.source(), pinnedSums, one.path, one.name, one.kind)
		if err != nil {
			return 0
		}
		info, err := os.Stat(at)
		if err != nil {
			return 0
		}
		total += info.Size()
	}
	return total
}
