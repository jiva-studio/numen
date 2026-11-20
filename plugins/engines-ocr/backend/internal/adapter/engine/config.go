package engine

import (
	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
)

func getThreads(cfg settings.Config) int {
	return onnx.LimitThreads(cfg.GetThreads(onnx.GetDefaultThreads()), cfg.Compute.ThreadLimit)
}
