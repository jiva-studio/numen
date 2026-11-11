package engine

import (
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine/onnx"
)

// The files one Parakeet export is published as.
const (
	encoderFile = "encoder.int8.onnx"
	decoderFile = "decoder.int8.onnx"
	joinerFile  = "joiner.int8.onnx"
	tokensFile  = "tokens.txt"
)

// sessionThreads is the intra-op thread count of each session. The encoder
// runs on whole stretches; the decoder, joiner and segmenter run on one small
// tensor at a time and take one thread.
type sessionThreads struct {
	encoder, decoder, joiner, segmenter int
}

func getSessionThreads(cfg settings.Config) sessionThreads {
	return sessionThreads{encoder: cfg.GetThreads(onnx.GetDefaultThreads()), decoder: 1, joiner: 1, segmenter: 1}
}
