package engine

import (
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/onnx"
)

func getThreads(cfg settings.Config) int {
	return cfg.GetThreads(onnx.GetDefaultThreads())
}
