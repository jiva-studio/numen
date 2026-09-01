package port

import (
	"context"
	"fmt"
)

// Transcription names what listened to a recording. It is recorded beside what
// it produced, as a Recognition is recorded with a reading: a text kept beyond
// the run that made it is claimed again by what made it.
//
// The segmenter and the threshold it cut at are part of the name. They decide
// where one stretch of speech ends, and a stretch cut elsewhere is transcribed
// into other words.
type Transcription struct {
	// Model is what turns speech into words.
	Model string
	// Segmenter is what finds the speech in the silence, and Cutting is every
	// setting it cuts by, as one value. Where a stretch of speech ends decides
	// what words come out of it, so all of them are named.
	Segmenter string
	Cutting   string

	// From is where the models were loaded from. Two models answering to one
	// name from two places are two models.
	From string
}

// String is the identity as one value, for saying what is listening.
func (t Transcription) String() string {
	return fmt.Sprintf("%s+%s", t.Model, t.Segmenter)
}

// Recipe is everything about this transcription that decides what a text is, as
// one value. It is kept beside an artifact so that a person can ask what heard
// the words they are reading.
func (t Transcription) Recipe() string {
	return fmt.Sprintf("%s|%s|%s|%s", t.From, t.Model, t.Segmenter, t.Cutting)
}

// Audio is one stretch of a recording that carries words, as the samples a
// model is given, and the milliseconds it spans. The samples are 16 kHz mono,
// which is what a model takes.
type Audio struct {
	Samples []float32
	From    int
	To      int
}

// A Recording is a file opened for listening: how long it is, and where the
// speech in it is.
//
// The length is answered before anything is listened to, so a person watching a
// transcription is told how much of it is left.
type Recording interface {
	// Length is how long the recording is, in milliseconds.
	Length() int

	// Speech is the stretches of speech from a millisecond onward, at most
	// count of them. A recording with nothing further to say answers with none,
	// and that is how a run knows it is done.
	Speech(ctx context.Context, from, count int) ([]Audio, error)

	Close() error
}

// Transcriber turns speech into the words it carries. The core asks for one and
// does not know whether the models run on this machine or a service answered.
type Transcriber interface {
	// Transcription is what every recording this transcriber hears was heard
	// by.
	Transcription() Transcription

	// Open is a recording, ready to be listened to. The bytes are the file as
	// the vault holds it, in whatever container it was recorded in.
	Open(ctx context.Context, raw []byte) (Recording, error)

	// Hear is one stretch of speech, as the words it carries. A stretch that
	// carries none is silence and not an error.
	Hear(ctx context.Context, audio Audio) (string, error)

	// Close releases whatever the models hold.
	Close() error
}
