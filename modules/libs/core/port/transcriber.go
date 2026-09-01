package port

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/transcript"
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
	// Segmenter is what finds the speech in the silence, and Threshold is how
	// sure it has to be.
	Segmenter string
	Threshold float32

	// From is where the models were loaded from. Two models answering to one
	// name from two places are two models.
	From string
}

// String is the identity as one value, for saying what is listening.
func (t Transcription) String() string {
	return fmt.Sprintf("%s+%s@%.2f", t.Model, t.Segmenter, t.Threshold)
}

// Recipe is everything about this transcription that decides what a text is, as
// one value. It is kept beside an artifact so that a person can ask what heard
// the words they are reading.
func (t Transcription) Recipe() string {
	return fmt.Sprintf("%s|%s|%s|%.2f", t.From, t.Model, t.Segmenter, t.Threshold)
}

// A Said is one stretch of speech: what was heard, and when it was said.
type Said struct {
	Text   string
	FromMs int
	ToMs   int
}

// Sound is a recording opened for listening: how long it is, and its speech in
// the order it was spoken.
//
// The length is answered before anything is listened to, because a person
// watching a transcription wants to know how much of it is left.
type Sound interface {
	// Length is how long the recording is, in milliseconds.
	Length() int

	// Speech is the stretches of speech from a millisecond onward, at most
	// count of them. A recording with nothing further to say answers with none.
	Speech(ctx context.Context, fromMs, count int) ([]Speech, error)

	Close() error
}

// A Speech is one stretch of a recording that carries words, as the samples a
// model is given. The samples are 16 kHz mono, which is what a model takes.
type Speech struct {
	Samples []float32
	FromMs  int
	ToMs    int
}

// Transcriber turns speech into the words it carries. The core asks for one and
// does not know whether the models run on this machine or a service answered.
type Transcriber interface {
	// Transcription is what every recording this transcriber hears was heard
	// by.
	Transcription() Transcription

	// Open is a recording, ready to be listened to. The bytes are the file as
	// the vault holds it, in whatever container it was recorded in.
	Open(ctx context.Context, raw []byte) (Sound, error)

	// Hear is one stretch of speech, as the words it carries. A stretch that
	// carries none is silence and not an error.
	Hear(ctx context.Context, speech Speech) (string, error)

	// Close releases whatever the models hold.
	Close() error
}

// Moments is where each of these stretches of speech sits in a text beginning
// at an offset. It is the one place the two are put together, so that a
// transcript and the moments beside it are cut from one run of arithmetic.
func Moments(said []Said, at int) []transcript.Moment {
	out := make([]transcript.Moment, 0, len(said))
	for _, one := range said {
		if one.Text == "" {
			continue
		}
		out = append(out, transcript.Moment{
			Start:  at,
			Length: len(one.Text),
			FromMs: one.FromMs,
			ToMs:   one.ToMs,
		})
		at += len(one.Text)
	}
	return out
}
