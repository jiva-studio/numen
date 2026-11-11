package engine

import (
	"os"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/ortraw"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/onnx"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/settings"
)

// Runtime is the ONNX Runtime a recording is transcribed through: the library, and the entry points
// the decoder and the segmenter call it by directly. A process makes one.
type Runtime struct {
	library *onnx.Library
	runner  *ortraw.Runner
}

// NewRuntime returns a runtime that is not open yet.
func NewRuntime() Runtime {
	runner := ortraw.NewRunner()
	return Runtime{library: onnx.NewLibrary(runner.Bind), runner: runner}
}

// Ready reports whether everything a transcription needs is already on this machine.
func Ready(runtime Runtime, cfg settings.Config) bool {
	if !runtime.library.IsAvailable(onnx.ResolveLibrary(cfg.Runtime, cfg.Dir)) {
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
func GetModelBytes(cfg settings.Config) int64 {
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
		// gomutants:disable-next-line BRANCH_IF reason="an empty path from locateModel fails os.Stat on the next line"
		if err != nil {
			return 0
		}
		info, err := os.Stat(pos)
		// gomutants:disable-next-line BRANCH_IF reason="locateModel already verified the file with os.Stat"
		if err != nil {
			return 0
		}
		total += info.Size()
	}
	return total
}
