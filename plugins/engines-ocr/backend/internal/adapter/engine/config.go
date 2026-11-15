package engine

import (
	"strings"

	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
)

func getThreads(cfg settings.Config) int {
	return cfg.GetThreads(onnx.GetDefaultThreads())
}

func getSessionProvider() onnx.Provider {
	if name := onnx.GetProvider(); !strings.EqualFold(strings.TrimSpace(string(name)), string(onnx.DefaultProvider)) {
		return name
	}
	return onnx.ProviderCPU
}
