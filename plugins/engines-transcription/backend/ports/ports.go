// Package ports declares the interfaces the use cases of the plugin share.
package ports

import (
	"context"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/speech"
)

// Recording is a file opened for listening.
type Recording interface {
	GetDuration() int
	GetSegments(ctx context.Context, from, count int) ([]speech.Audio, error)
	Close() error
}

// Recognizer turns the speech of a recording into words.
type Recognizer interface {
	Open(ctx context.Context, raw []byte) (Recording, error)
	Transcribe(ctx context.Context, audio speech.Audio) (string, error)
}
