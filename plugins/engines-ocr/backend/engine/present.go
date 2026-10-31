package engine

import (
	"os"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

// IsFetched reports whether one recogniser model file is on this machine.
func IsFetched(cfg Config, model RecogniserModel) bool {
	if model.Path != "" {
		return isPresent(model.Path)
	}
	_, err := onnx.LocateModel("", "NUMEN_DEPENDENCY_PP_OCR_REC", "pp-ocr-rec.onnx", cfg.Dir)
	return err == nil
}

func isPresent(pos string) bool {
	_, err := os.Stat(pos)
	return err == nil
}
