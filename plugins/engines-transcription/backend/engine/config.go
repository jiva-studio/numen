package engine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/segmenter"
)

// Config is what a user may change about transcription: which models, where
// they came from, how the speech in a file is found, and how much of the
// machine one recording may use.
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

	// Language is the language the settings name for the model: one of its Languages, or empty.
	Language string `json:"language"`

	Model     ParakeetModel   `json:"model"`
	Segmenter segmenter.Model `json:"segmenter"`

	// Progress receives the download progress when a listener is set. It is not a
	// setting and is not persisted: it carries progress to the observer.
	Progress func(what string, done, total int64) `json:"-"`
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

	// Languages are the languages the model can be set to transcribe, LanguageDetected where it detects
	// the language itself.
	Languages []string `json:"-"`
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
		Runtime:  "",
		Dir:      "",
		Threads:  0,
		Workers:  0,
		Language: "",
		Progress: nil,
		Model:    models[0],
		Segmenter: segmenter.Model{
			Name:      "silero-vad",
			Repo:      segmenterDefault,
			Path:      "",
			Threshold: 0,
			Silence:   0,
			Pad:       0,
			Longest:   0,
			Shortest:  0,
			Least:     0,
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

// checkLanguage reports a language the settings name that the model does not transcribe.
func (c Config) checkLanguage() error {
	if c.Language == "" || slices.Contains(c.Model.Languages, c.Language) {
		return nil
	}
	return fmt.Errorf("%s transcribes %s, and the settings name %q", c.Model.Name, strings.Join(c.Model.Languages, ", "), c.Language)
}
