package engine

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

// Ready reports whether everything needed to recognise a page is on this
// machine.
func Ready(cfg Config) bool {
	if !onnx.IsAvailable(onnx.ResolveLibrary(cfg.Runtime, cfg.Dir)) {
		return false
	}
	if _, err := onnx.LocateModel(cfg.Layout.Path, "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", cfg.Dir); err != nil {
		return false
	}
	if _, err := onnx.LocateModel(cfg.Detect.Path, "NUMEN_DEPENDENCY_PP_OCR_DET", "pp-ocr-det.onnx", cfg.Dir); err != nil {
		return false
	}
	if _, err := onnx.LocateModel(cfg.Recognise.Path, "NUMEN_DEPENDENCY_PP_OCR_REC", "pp-ocr-rec.onnx", cfg.Dir); err != nil {
		return false
	}
	return true
}
