package engine

import (
	"runtime"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

func getProvider(m model.LocalModel) onnx.Provider {
	// A model that names no provider runs on the one the host chose, which is auto when it chose none.
	name := m.Provider
	if name == "" {
		name = string(onnx.GetProvider())
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "cpu":
		return onnx.ProviderCPU
	case "coreml", "core-ml":
		return onnx.ProviderCoreML
	case "cuda":
		return onnx.ProviderCUDA
	case "directml", "direct-ml", "dml":
		return onnx.ProviderDirectML
	case "rocm":
		return onnx.ProviderROCm
	}
	return onnx.ProviderAuto
}

func getThreads(m model.LocalModel) int {
	if m.Threads > 0 {
		return m.Threads
	}
	return onnx.GetDefaultThreads()
}

const (
	passLadderLow  = 2
	passLadderHigh = 4
)

func getPasses(m model.LocalModel, provider onnx.Provider) int {
	switch {
	case m.Sessions > 0:
		return m.Sessions
	case provider != onnx.ProviderCPU:
		return 1
	case m.PassBudget > 0:
		return m.PassBudget
	}
	// gomutants:disable-next-line INTEGER_DECREMENT reason="GOMAXPROCS reads the setting for every argument below one"
	cpus := runtime.GOMAXPROCS(0)
	switch {
	case cpus <= passLadderLow:
		return 1
	case cpus <= passLadderHigh:
		return 2
	}
	return 3
}
