package engine

import "context"

// Audio is one stretch of a recording that carries words, as the samples a
// model is given, and the milliseconds it spans. The samples are 16 kHz mono.
type Audio struct {
	Samples []float32
	From    int
	To      int
}

// TranscriptionModel names what listened to a recording.
type TranscriptionModel struct {
	Model     string
	Segmenter string
	Cutting   string
	From      string
}

// A Recording is a file opened for listening.
type Recording interface {
	GetDuration() int
	GetSegments(ctx context.Context, from, count int) ([]Audio, error)
	Close() error
}
