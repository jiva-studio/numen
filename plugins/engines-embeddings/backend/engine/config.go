package engine

import (
	"runtime"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/onnx"
)

// getProvider resolves the onnxruntime Provider. A model that names none runs
// on the provider the host chose, which is auto when it chose none.
func getProvider(m model.LocalModel) onnx.Provider {
	name := m.Provider
	if name == "" {
		name = string(onnx.GetProvider())
	}
	p, err := onnx.ParseProvider(name)
	if err != nil {
		return onnx.ProviderAuto
	}
	return p
}

// getThreads resolves the thread count.
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

// getPasses calculates the concurrency level.
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
