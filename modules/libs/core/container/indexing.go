package container

import (
	"math"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
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

	// Proofreading is what puts a recognition right. Naming no profile here is
	// naming no proofreader, and a recognition is used as it was read.
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
// recognition right afterwards.
type Recognition struct {
	RecognitionConfig

	// Proofread names the profile a recognition is put right at. Automatically
	// there says whether a recognition just made is put right without anybody
	// asking.
	Proofread proofreading.Proofread `json:"proofread"`
}

// RecognitionConfig holds the configuration for optical character recognition.
type RecognitionConfig struct {
	// Runtime is the shared library of ONNX runtime. Empty means the one beside the application.
	Runtime string `json:"runtime"`

	// Dir is a folder holding the models. Empty means the folder beside the
	// application, and then the download cache.
	Dir string `json:"dir"`

	// ShouldDownload allows fetching what is not on this machine.
	ShouldDownload bool `json:"download"`

	Layout    LayoutModel     `json:"layout"`
	Detect    DetectModel     `json:"detect"`
	Recognise RecogniserModel `json:"recognise"`
	Regions   RegionKinds     `json:"regions"`
}

// LayoutModel divides a page into its parts and puts them in reading order.
type LayoutModel struct {
	// Name is where the model is fetched from, and Path is a file on this
	// machine. A path is used as given; a name is looked for in Dir first.
	Name string `json:"name"`
	// Path is this model as a file on this machine.
	Path string `json:"path"`

	// Labels are the parts the model knows, in the order of its class ids.
	Labels []string `json:"labels"`

	// Minimum is the score a part carries to be a part at all.
	Minimum float32 `json:"minimum"`
	// Overlap is how much of two parts may be common before the second is taken
	// to be the first found again.
	Overlap float64 `json:"overlap"`
	// Margin is what a part is widened by before it is read, because the first
	// letter of a line sits on the boundary the model drew.
	Margin int `json:"margin"`
}

// DetectModel finds the lines one part holds.
type DetectModel struct {
	// Name is where this model is fetched from.
	Name string `json:"name"`
	// Path is this model as a file on this machine.
	Path string `json:"path"`

	// MaxSide is the longest side a part is read at.
	MaxSide int `json:"max_side"`
	// Expand is how many pixels a found line is widened by, in the image the
	// detector reads: the part scaled so that its longest side is MaxSide.
	Expand int `json:"expand"`
	// Minimum is the heat a pixel carries to be part of a line.
	Minimum float32 `json:"minimum"`
}

// RecogniserModel reads what a line says: which model, at what size a page and
// a line reach it, and how much of the machine it takes.
type RecogniserModel struct {
	// Name is where this model is fetched from.
	Name string `json:"name"`
	// Path is this model as a file on this machine.
	Path string `json:"path"`

	// Dict is the model's characters, one to a line. Empty is the ordinary
	// case: the model carries them, and asking it is the only way to be sure.
	Dict string `json:"dict"`
	// Classes is how many characters the model knows and two more. Zero asks
	// the model. A number that disagrees with the model is refused.
	Classes int64 `json:"classes"`

	// DPI is what a page is rendered at.
	DPI int `json:"dpi"`
	// Height is what a line is scaled to before it is read.
	Height int `json:"height"`

	// Jobs is how many recognition sessions run at once. Zero takes the performance profile.
	Jobs int `json:"jobs"`
	// Threads is how many threads one model may use. Zero takes the performance profile.
	Threads int `json:"threads"`
}

// RegionKinds says what the parts of a page are for. A part the model names
// that Body does not carry is not read.
type RegionKinds struct {
	// Body carry what the document says.
	Body []string `json:"body"`
	// Head open a part of the document, outermost first: where a name stands in
	// the list is how deep the part it opens sits. Each of them is read only
	// where Body carries it too.
	Head []string `json:"head"`
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
		Embedding: embed.Defaults(),
		Recognition: Recognition{
			RecognitionConfig: RecognitionConfig{
				Runtime:        "",
				Dir:            "",
				ShouldDownload: true,
				Layout: LayoutModel{
					Name:    "",
					Path:    "",
					Labels:  nil,
					Minimum: 0,
					Overlap: 0,
					Margin:  0,
				},
				Detect: DetectModel{
					Name:    "",
					Path:    "",
					MaxSide: 0,
					Expand:  0,
					Minimum: 0,
				},
				Recognise: RecogniserModel{
					Name:    "",
					Path:    "",
					Dict:    "",
					Classes: 0,
					DPI:     300,
					Height:  0,
					Jobs:    0,
					Threads: 0,
				},
				Regions: RegionKinds{
					Body: nil,
					Head: nil,
				},
			},
			Proofread: proofreading.Proofread{
				Profile:     "",
				IsAutomatic: false,
			},
		},
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
