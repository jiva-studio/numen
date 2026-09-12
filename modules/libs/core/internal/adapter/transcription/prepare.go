package transcription

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// Ready says whether everything a transcription needs is already on this
// machine. It opens nothing and fetches nothing, so it is answered while a
// transcription is fetching what it needs.
func Ready(cfg Config) bool {
	cfg.Download = false
	if !onnxruntime.Here(cfg.settings()) {
		return false
	}
	var found paths
	for _, one := range getWantedFiles(cfg, &found) {
		if _, err := model(context.Background(), cfg, one.path, one.name, one.kind); err != nil {
			return false
		}
	}
	return true
}
