package engine

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
	"github.com/jiva-studio/numen/modules/plugins/ocr/backend/onnxruntime"
)

// Ready says whether everything recognising a page needs is already on this machine.
func Ready(cfg Config) bool {
	cfg.ShouldDownload = false
	if !onnxruntime.IsAvailable(cfg.getRuntimeLibrary()) {
		return false
	}
	for _, one := range []struct{ path, name, what string }{
		{cfg.Layout.Path, cfg.Layout.Name, "layout model"},
		{cfg.Detect.Path, cfg.Detect.Name, "detect model"},
		{cfg.Recognise.Path, cfg.Recognise.Name, "recognise model"},
	} {
		if _, err := models.Locate(context.Background(), cfg.source(), pinnedSums, one.path, one.name, one.what); err != nil {
			return false
		}
	}
	return true
}
