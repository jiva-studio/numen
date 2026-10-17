package engine

import (
	"context"

	"github.com/jiva-studio/numen/modules/plugins/ocr/backend/onnxruntime"
)

// Ready says whether everything recognising a page needs is already on this machine.
func Ready(cfg Config) bool {
	cfg.ShouldDownload = false
	if !onnxruntime.IsAvailable(cfg.Runtime) {
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
