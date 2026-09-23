package recognition

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// Ready says whether everything reading a page needs is already on this
// machine. It opens nothing and fetches nothing, so it is answered while a
// reading is fetching what it needs.
func Ready(cfg Config) bool {
	cfg.ShouldDownload = false
	if !onnxruntime.IsHere(cfg.settings()) {
		return false
	}
	for _, one := range []struct{ path, name, what string }{
		{cfg.Layout.Path, cfg.Layout.Name, "layout"},
		{cfg.Detect.Path, cfg.Detect.Name, "detect"},
		{cfg.Recognise.Path, cfg.Recognise.Name, "recognise"},
	} {
		if _, err := model(context.Background(), cfg, one.path, one.name, one.what); err != nil {
			return false
		}
	}
	return true
}
