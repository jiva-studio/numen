package engine

import (
	"os"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

// Ready reports whether everything a transcription needs is already on this machine.
func Ready(cfg Config) bool {
	if !onnx.IsAvailable(onnx.ResolveLibrary(cfg.Runtime, cfg.Dir)) {
		return false
	}
	if _, err := locateModel(cfg.Model.Encoder, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_ENCODER", encoderFile, cfg.Dir); err != nil {
		return false
	}
	if _, err := locateModel(cfg.Model.Decoder, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_DECODER", decoderFile, cfg.Dir); err != nil {
		return false
	}
	if _, err := locateModel(cfg.Model.Joiner, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_JOINER", joinerFile, cfg.Dir); err != nil {
		return false
	}
	if _, err := locateModel(cfg.Model.Tokens, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_TOKENS", tokensFile, cfg.Dir); err != nil {
		return false
	}
	if _, err := locateModel(cfg.Segmenter.Path, cfg.Segmenter.Repo, "NUMEN_DEPENDENCY_SILERO_VAD", "silero.onnx", cfg.Dir); err != nil {
		return false
	}
	return true
}

// GetModelBytes is the size of the model files on this machine, and zero where any of them is not here.
func GetModelBytes(cfg Config) int64 {
	targets := []struct{ explicit, repo, envVar, fileName string }{
		{cfg.Model.Encoder, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_ENCODER", encoderFile},
		{cfg.Model.Decoder, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_DECODER", decoderFile},
		{cfg.Model.Joiner, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_JOINER", joinerFile},
		{cfg.Model.Tokens, cfg.Model.Repo, "NUMEN_DEPENDENCY_PARAKEET_TOKENS", tokensFile},
		{cfg.Segmenter.Path, cfg.Segmenter.Repo, "NUMEN_DEPENDENCY_SILERO_VAD", "silero.onnx"},
	}
	var total int64
	for _, t := range targets {
		pos, err := locateModel(t.explicit, t.repo, t.envVar, t.fileName, cfg.Dir)
		if err != nil {
			return 0
		}
		info, err := os.Stat(pos)
		if err != nil {
			return 0
		}
		total += info.Size()
	}
	return total
}
