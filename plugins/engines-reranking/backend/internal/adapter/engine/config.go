package engine

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/shared/onnx"
)

// batchPairs is the most pairs one pass of the model reads.
const batchPairs = 8

func getSessionSettings(modelPath, provider string, threadLimit int) onnx.SessionSettings {
	return onnx.SessionSettings{
		Provider:     onnx.GetSessionProvider(provider),
		Threads:      onnx.LimitThreads(onnx.GetDefaultThreads(), threadLimit),
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
