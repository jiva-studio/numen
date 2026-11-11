package engine

import (
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/settings"
)

func getThreads(cfg settings.Config) int {
	return cfg.GetThreads(onnx.GetDefaultThreads())
}
