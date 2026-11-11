package engine

import (
	"os"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
)

// IsFetched reports whether one recognizer model file is on this machine.
func IsFetched(cfg settings.Config, model settings.RecognizerModel) bool {
	if model.Path != "" {
		return isPresent(model.Path)
	}
	_, err := locateModel("", "NUMEN_DEPENDENCY_PP_OCR_REC", "pp-ocr-rec.onnx", cfg.Dir)
	return err == nil
}

func isPresent(pos string) bool {
	_, err := os.Stat(pos)
	return err == nil
}
