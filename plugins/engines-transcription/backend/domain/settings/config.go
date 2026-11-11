package settings

// Config is what a user may change about transcription: which models, where they came from, how the speech in a file is found, and how much of the machine one recording may use.
// A zero field takes its default, so a settings file naming one thing leaves the rest alone.
type Config struct {
	Runtime string `json:"runtime"`

	Dir string `json:"dir"`

	ShouldDownload bool `json:"download"`

	Threads int `json:"threads"`

	Workers int `json:"-"`

	Model     ParakeetModel  `json:"model"`
	Segmenter SegmenterModel `json:"segmenter"`

	Progress func(what string, done, total int64) `json:"-"`
}

// ParakeetModel turns speech into words. It is exported as three graphs and the
// pieces they write, and all four are one model: three graphs from two exports
// answer with nothing anybody can read.
type ParakeetModel struct {
	Name string `json:"name"`
	Repo string `json:"from"`

	Encoder string `json:"encoder"`
	Decoder string `json:"decoder"`
	Joiner  string `json:"joiner"`
	Tokens  string `json:"tokens"`
}

// Defaults listen with parakeet-tdt-0.6b-v3 and find the speech with
// silero-vad.
func Defaults() Config {
	return Config{
		Runtime:  "",
		Dir:      "",
		Threads:  0,
		Workers:  0,
		Progress: nil,
		Model:    GetModels()[0],
		Segmenter: SegmenterModel{
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

// GetThreads returns the thread count of one recording; cores is the count taken when none is set.
func (c Config) GetThreads(cores int) int {
	if c.Threads <= 0 {
		return cores
	}
	return c.Threads
}
