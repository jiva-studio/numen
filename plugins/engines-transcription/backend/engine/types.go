package engine

import (
	"context"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcribe"
)

// Audio is one stretch of a recording that carries words, as the samples a
// model is given, and the milliseconds it spans. The samples are 16 kHz mono.
type Audio = transcribe.Audio

// A Recording is a file opened for listening.
type Recording = transcribe.Recording

// TranscriberPort is the port interface for speech recognition and segmentation.
type TranscriberPort interface {
	Open(ctx context.Context, raw []byte) (Recording, error)
	Transcribe(ctx context.Context, audio Audio) (string, error)
	Close() error
}
