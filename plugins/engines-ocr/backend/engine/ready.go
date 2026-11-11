package engine

import (
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/settings"
)

// Ready reports whether everything needed to recognize a page is on this
// machine.
func Ready(library *onnx.Library, cfg settings.Config) bool {
	if !library.IsAvailable(onnx.ResolveLibrary(cfg.Runtime, cfg.Dir)) {
		return false
	}
	if _, err := locateModel(cfg.Layout.Path, "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", cfg.Dir); err != nil {
		return false
	}
	if _, err := locateModel(cfg.Detect.Path, "NUMEN_DEPENDENCY_PP_OCR_DET", "pp-ocr-det.onnx", cfg.Dir); err != nil {
		return false
	}
	if _, err := locateModel(cfg.Recognize.Path, "NUMEN_DEPENDENCY_PP_OCR_REC", "pp-ocr-rec.onnx", cfg.Dir); err != nil {
		return false
	}
	return true
}
