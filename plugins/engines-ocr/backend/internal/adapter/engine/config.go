package engine

import (
	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
)

func getThreads(cfg settings.Config) int {
	return cfg.GetThreads(onnx.GetDefaultThreads())
}
