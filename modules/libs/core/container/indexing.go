package container

import (
	"math"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/transcription"
)

// Indexing is how a vault is made searchable. It is one section of the settings
// file and the union of what four adapters are configured with, which is why it
// stands where every adapter is bound.
type Indexing struct {
	// Embedding is which model turns text into vectors, and how it is reached.
	Embedding embed.Config `json:"embedding"`

	// Recognition is how a scanned document is read when a person asks for it.
	// Nothing here runs on its own.
	Recognition Recognition `json:"recognition"`

	// Proofreading is what puts a reading right. Naming no profile here is
	// naming no proofreader, and a reading is used as it was read.
	Proofreading proofreading.Config `json:"proofreading"`

	// Transcription is how a recording is listened to: which models hear it,
	// where they came from, and how the speech in it is found.
	Transcription Transcription `json:"transcription"`

	// TranscribeRecordings is whether a recording the vault holds no transcript
	// for is listened to without anybody asking. A file leaving it out listens
	// to them, and a file naming false leaves it to the hand. A vault of a
	// hundred hours is a day of a machine, and how much of it to spend is the
	// person's.
	TranscribeRecordings *bool `json:"transcribe_recordings"`

	// TranscribeUnderMB is how large a recording may be and still be listened
	// to without anybody asking, in megabytes. A larger one waits to be asked
	// for by name, because a folder of albums is days of a machine and nobody
	// put them there to be read.
	//
	// Zero takes the default. A negative number is no limit at all.
	TranscribeUnderMB int `json:"transcribe_under_mb"`
}

// Recognition is how a scanned document is read, and which profile puts that
// reading right afterwards.
type Recognition struct {
	recognition.Config

	// Proofread names the profile a reading is put right at. Automatically
	// there says whether a reading just made is put right without anybody
	// asking.
	Proofread proofreading.Proofread `json:"proofread"`
}

// Transcription is how a recording is listened to, and which profile puts what
// was heard right afterwards.
type Transcription struct {
	transcription.Config

	// Proofread names the profile a transcript is put right at. Automatically
	// there says whether a transcript already written down is put right
	// without anybody asking; whether a recording nobody asked about is
	// listened to at all is TranscribeRecordings.
	Proofread proofreading.Proofread `json:"proofread"`
}

// DefaultIndexing is what an installation nobody has configured makes a vault
// searchable with.
func DefaultIndexing() Indexing {
	set := true
	return Indexing{
		Embedding:            embed.Defaults(),
		Recognition:          Recognition{Config: recognition.Defaults()},
		Proofreading:         proofreading.Defaults(),
		Transcription:        Transcription{Config: transcription.Defaults()},
		TranscribeRecordings: &set,
	}
}

// DefaultTranscribeUnderMB is how large a recording listened to unasked may be.
// It is a talk of a few hours at the bitrates a recorder writes, and larger than
// anything a person speaks into a phone.
const DefaultTranscribeUnderMB = 300

// CanTranscribe is whether a recording is listened to without being asked. A
// section naming nothing listens to them.
func (i Indexing) CanTranscribe() bool {
	return i.TranscribeRecordings == nil || *i.TranscribeRecordings
}

// MostTranscribeUnderMB is as many megabytes as a size in bytes reaches. A
// larger number names no limit any recording could pass.
const MostTranscribeUnderMB = math.MaxInt64 >> 20

// GetTranscribeLimit is how many bytes a recording may run to and still be
// listened to unasked. A negative setting is no limit.
func (i Indexing) GetTranscribeLimit() int64 {
	switch {
	case i.TranscribeUnderMB < 0:
		return 0
	case i.TranscribeUnderMB == 0:
		return DefaultTranscribeUnderMB << 20
	case i.TranscribeUnderMB > MostTranscribeUnderMB:
		return math.MaxInt64
	}
	return int64(i.TranscribeUnderMB) << 20
}
