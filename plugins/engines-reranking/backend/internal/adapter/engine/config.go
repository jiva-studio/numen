package engine

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/shared/onnx"
)

// batchPairs is the most pairs one pass of the model reads.
const batchPairs = 8

func getSessionSettings(modelPath string) onnx.SessionSettings {
	return onnx.SessionSettings{
		Provider:     onnx.GetSessionProvider(),
		Threads:      onnx.GetDefaultThreads(),
		ComputeUnits: onnx.ComputeUnitsDefault,
		CacheRoot:    getCacheRoot(),
		ModelPath:    modelPath,
	}
}

func getCacheRoot() string {
	root, err := cache.GetSharedDirOrPluginDir()
	if err != nil {
		return ""
	}
	return root
}
