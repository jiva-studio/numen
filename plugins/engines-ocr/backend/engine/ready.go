package engine

import (
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/onnx"
)

// Ready reports whether everything needed to recognise a page is on this
// machine.
func Ready(library *onnx.Library, cfg Config) bool {
	if !library.IsAvailable(onnx.ResolveLibrary(cfg.Runtime, cfg.Dir)) {
		return false
	}
	if _, err := locateModel(cfg.Layout.Path, "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", cfg.Dir); err != nil {
		return false
	}
	if _, err := locateModel(cfg.Detect.Path, "NUMEN_DEPENDENCY_PP_OCR_DET", "pp-ocr-det.onnx", cfg.Dir); err != nil {
		return false
	}
	_, err := locateRecognisers(cfg)
	return err == nil
}
