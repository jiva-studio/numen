package transcription

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
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

	// Download allows fetching what is not on this machine.
	Download bool `json:"download"`

	// Threads is how many threads one model may use.
	Threads int `json:"threads"`

	Model  ParakeetModel `json:"model"`
	Speech SegmenterModel   `json:"speech"`

	// Fetching is told how far a download has got, when anything is listening.
	// It is not a setting and is not written down: it is how the wait reaches
	// whoever is watching it.
	Fetching func(what string, done, total int64) `json:"-"`
}

// settings are what the runtime and the models are found by.
func (c Config) settings() onnxruntime.Settings {
	return onnxruntime.Settings{
		Section:  "transcription",
		Runtime:  c.Runtime,
		Dir:      c.Dir,
		Download: c.Download,
		Fetching: c.Fetching,
	}
}

// ParakeetModel turns speech into words. It is exported as three graphs and the
// pieces they write, and all four are one model: three graphs from two exports
// answer with nothing anybody can read.
type ParakeetModel struct {
	// Name is what this model is called in the record kept beside a text.
	Name string `json:"name"`
	// From is the folder the four files are fetched from.
	From string `json:"from"`

	// Encoder, Decoder, Joiner and Tokens are the files on this machine. A path
	// is used as given; an empty one is the file of that name under From.
	Encoder string `json:"encoder"`
	Decoder string `json:"decoder"`
	Joiner  string `json:"joiner"`
	Tokens  string `json:"tokens"`
}

// SegmenterModel finds where in a recording somebody is speaking. What it cuts is
// what the transcriber is given, so where it cuts is part of what the words
// are.
type SegmenterModel struct {
	// Name is what this segmenter is called in the record kept beside a text.
	Name string `json:"name"`
	// From is where the model is fetched from, and Path is a file on this
	// machine. A path is used as given; From is looked for in Dir first.
	From string `json:"from"`
	Path string `json:"path"`

	// Threshold is how sure the model has to be that a window carries speech.
	Threshold float32 `json:"threshold"`
	// Silence is how much quiet, in milliseconds, closes a stretch of speech.
	Silence int `json:"silence"`
	// Pad is how many milliseconds are kept on each side of a stretch, so that
	// the first and last sound of a word are inside it.
	Pad int `json:"pad"`
	// Longest is how many milliseconds one stretch may run to. Speech that goes
	// on longer is cut at the quietest window this side of the limit.
	Longest int `json:"longest"`
	// Shortest is how many milliseconds a stretch carries to be a stretch at
	// all.
	Shortest int `json:"shortest"`
	// Least is how many milliseconds a stretch runs to before it stands as a
	// line of its own. A shorter one is put together with the stretch after it.
	Least int `json:"least"`
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
			From: "https://huggingface.co/csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8/resolve/main/",
		},
		Speech: SegmenterModel{
			Name: "silero-vad",
			From: "https://huggingface.co/onnx-community/silero-vad/resolve/main/onnx/model.onnx",
		},
		Threads: 4,

		// What a transcription needs is fetched when it is wanted.
		Download: true,
	}
}

// The defaults for everything a settings file leaves out.
func (c Config) threads() int {
	if c.Threads <= 0 {
		return 4
	}
	return c.Threads
}

func (s SegmenterModel) threshold() float32 {
	if s.Threshold <= 0 {
		return 0.5
	}
	return s.Threshold
}

func (s SegmenterModel) silence() int {
	if s.Silence <= 0 {
		return 500
	}
	return s.Silence
}

// The model answers on the window a sound begins in, and the sound before that
// window is what the first letter of the word is made of.
func (s SegmenterModel) pad() int {
	if s.Pad <= 0 {
		return 200
	}
	return s.Pad
}

// One stretch is one run of the encoder, and its cost grows with its length.
func (s SegmenterModel) longest() int {
	if s.Longest <= 0 {
		return 30000
	}
	return s.Longest
}

func (s SegmenterModel) shortest() int {
	if s.Shortest <= 0 {
		return 100
	}
	return s.Shortest
}

// A line of a transcript is read, so it holds a phrase and not a breath. A
// speaker hesitating in the middle of a sentence stops for about this long.
func (s SegmenterModel) least() int {
	if s.Least <= 0 {
		return 2500
	}
	return s.Least
}

// cutting is every setting a stretch of speech is cut by, as one value. Each of
// them moves where a stretch ends, and a stretch that ends elsewhere is heard
// as other words.
func (s SegmenterModel) cutting() string {
	return fmt.Sprintf("%.2f/%d/%d/%d/%d/%d",
		s.threshold(), s.silence(), s.pad(), s.longest(), s.shortest(), s.least())
}

// paths are the files this run listens through, and where they were found.
type paths struct {
	// engine is the ONNX Runtime this run loaded, and runtime is where it came
	// from.
	engine  *ort.Engine
	runtime string
	encoder string
	decoder string
	joiner  string
	tokens  string
	speech  string
	from    string
}

// locate finds the runtime and the models.
//
// Three places are tried in order and each is a setting: a path written down,
// the folder the application was installed into, and what was downloaded. A
// A path that is written down is used as given, and its absence is an error.
func locate(ctx context.Context, cfg Config) (paths, error) {
	found := paths{from: "settings"}

	var err error
	if found.engine, found.runtime, err = onnxruntime.Open(ctx, cfg.settings()); err != nil {
		return paths{}, err
	}
	for _, one := range wanted(cfg, &found) {
		if *one.into, err = model(ctx, cfg, one.path, one.name, one.what); err != nil {
			return paths{}, err
		}
	}
	if cfg.Dir != "" {
		found.from = cfg.Dir
	}
	return found, nil
}

// A wantedFile is one of the things a transcription reads: where it is put once
// it is found, where the settings say it is, where it is fetched from, and what
// it is called when it is missing.
type wantedFile struct {
	into             *string
	path, name, what string
}

// wanted is every file a transcription reads. The transducer is four of them,
// published as four names in one folder.
func wanted(cfg Config, into *paths) []wantedFile {
	under := func(name string) string {
		if cfg.Model.From == "" {
			return ""
		}
		return strings.TrimSuffix(cfg.Model.From, "/") + "/" + name
	}
	return []wantedFile{
		{&into.encoder, cfg.Model.Encoder, under(encoderFile), "encoder"},
		{&into.decoder, cfg.Model.Decoder, under(decoderFile), "decoder"},
		{&into.joiner, cfg.Model.Joiner, under(joinerFile), "joiner"},
		{&into.tokens, cfg.Model.Tokens, under(tokensFile), "tokens"},
		{&into.speech, cfg.Speech.Path, cfg.Speech.From, "speech model"},
	}
}

// model is where one model's file is.
//
// A path written down is used as given, and its absence is an error rather than
// a reason to look elsewhere: a person who said where a model is meant it. A
// name is looked for beside the application and then fetched.
func model(ctx context.Context, cfg Config, path, name, what string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("the %s: %w", what, err)
		}
		return path, nil
	}
	if name == "" {
		return "", fmt.Errorf("no %s: name one, or say where it is", what)
	}
	for _, at := range onnxruntime.Beside(cfg.Dir, filepath.Base(name)) {
		if _, err := os.Stat(at); err == nil {
			return at, nil
		}
	}
	if !onnxruntime.IsAddress(name) {
		return "", fmt.Errorf("the %s %q is not beside the application, and is not somewhere to fetch it from", what, name)
	}
	found, err := onnxruntime.Fetched(ctx, cfg.settings(), name)
	if err != nil {
		return "", fmt.Errorf("the %s: %w", what, err)
	}
	return found, nil
}

// named is what a model is called, for the record kept beside what it produced.
// A name in the settings stands, and a model without one is called after the
// file it was loaded from.
func named(name, at string) string {
	if name != "" {
		return name
	}
	return strings.TrimSuffix(path.Base(filepath.ToSlash(at)), filepath.Ext(at))
}
