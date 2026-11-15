package engine

import (
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine/onnx"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine/ortraw"
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
