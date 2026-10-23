package engine

import (
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
	"github.com/jiva-studio/numen/modules/plugins/engines.transcription/backend/engine/segmenter"
)

// Config is what a person may change about listening to a recording: which
// models, where they came from, how the speech in a file is found, and how much
// of the machine one recording may use.
//
// A zero field takes its default, so a settings file naming one thing leaves
// the rest alone.
type Config struct {
	// Runtime is the ONNX Runtime shared library. Empty means the one beside
	// the application, and then the one the platform holds.
	Runtime string `json:"runtime"`

	// Dir is a folder holding the models. Empty means the folder beside the
	// application, and then the download cache.
	Dir string `json:"dir"`

	// ShouldDownload allows fetching what is not on this machine.
	ShouldDownload bool `json:"download"`

	// Threads is how many threads the encoder may use. Zero takes one per core, up to eight.
	Threads int `json:"threads"`

	// Workers is how many goroutines share the resampling and feature extraction of a recording. It is not a setting: the composition root fills it from the compute plan, and zero takes one per core.
	Workers int `json:"-"`

	Model     ParakeetModel   `json:"model"`
	Segmenter segmenter.Model `json:"segmenter"`

	// Progress is told how far a download has got, when anything is listening.
	// It is not a setting and is not written down: it is how the wait reaches
	// whoever is watching it.
	Progress func(what string, done, total int64) `json:"-"`
}

// folder is where the models are kept and whether they may be fetched.
func (c Config) folder() models.Folder {
	return models.NewFolder("indexing.transcription", c.Dir, c.ShouldDownload, c.Progress)
}

// ParakeetModel turns speech into words. It is exported as three graphs and the
// pieces they write, and all four are one model: three graphs from two exports
// answer with nothing anybody can read.
type ParakeetModel struct {
	// Name is what this model is called in the record kept beside a text.
	Name string `json:"name"`
	// Repo is the folder the four files are fetched from.
	Repo string `json:"from"`

	// Encoder, Decoder, Joiner and Tokens are the files on this machine. A path
	// is used as given; an empty one is the file of that name under Repo.
	Encoder string `json:"encoder"`
	Decoder string `json:"decoder"`
	Joiner  string `json:"joiner"`
	Tokens  string `json:"tokens"`
}

// The files one Parakeet export is published as.
const (
	encoderFile = "encoder.int8.onnx"
	decoderFile = "decoder.int8.onnx"
	joinerFile  = "joiner.int8.onnx"
	tokensFile  = "tokens.txt"
)

// Defaults listen with parakeet-tdt-0.6b-v3 and find the speech with
// silero-vad.
func Defaults() Config {
	return Config{
		Model: ParakeetModel{
			Name: "parakeet-tdt-0.6b-v3-int8",
			Repo: parakeetDefault,
		},
		Segmenter: segmenter.Model{
			Name: "silero-vad",
			Repo: segmenterDefault,
		},

		// What a transcription needs is fetched when it is wanted.
		ShouldDownload: true,
	}
}

// The defaults for everything a settings file leaves out.
func (c Config) threads() int {
	if c.Threads <= 0 {
		return onnx.GetDefaultThreads()
	}
	return c.Threads
}

// sessionThreads is the intra-op thread count of each session. The encoder
// runs on whole stretches; the decoder, joiner and segmenter run on one small
// tensor at a time and take one thread.
type sessionThreads struct {
	encoder, decoder, joiner, segmenter int
}

func (c Config) sessionThreads() sessionThreads {
	return sessionThreads{encoder: c.threads(), decoder: 1, joiner: 1, segmenter: 1}
}
