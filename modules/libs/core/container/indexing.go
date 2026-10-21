package container

import (
	"encoding/json"
	"math"
)

// Indexing is how a vault is made searchable. It is one section of the settings
// file and the union of what four adapters are configured with, which is why it
// stands where every adapter is bound.
type Indexing struct {
	// Embedding is which model turns text into vectors, and how it is reached.
	Embedding EmbeddingConfig `json:"embedding"`

	// Recognition is how a scanned document is read when a person asks for it.
	// Nothing here runs on its own.
	Recognition Recognition `json:"recognition"`

	// Proofreading is what puts a recognition right. Naming no profile here is
	// naming no proofreader, and a recognition is used as it was read.
	Proofreading ProofreadingConfig `json:"proofreading"`

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
	// Proofread names the profile a recognition is put right at. Automatically
	// there says whether a recognition just made is put right without anybody
	// asking.
	Proofread Proofread `json:"proofread"`
}

// Transcription is how a recording is listened to, and which profile puts what
// was heard right afterwards.
type Transcription struct {
	// Proofread names the profile a transcript is put right at. Automatically
	// there says whether a transcript already written down is put right
	// without anybody asking; whether a recording nobody asked about is
	// listened to at all is TranscribeRecordings.
	Proofread Proofread `json:"proofread"`
}

// ProofreadingConfig is what puts a recognition right.
type ProofreadingConfig struct {
	// MaxEditDistance is how far a correction may stand from the line as read
	// and still be a correction: the Levenshtein distance between their letters,
	// as a share of the longer of the two. Zero leaves the limit to the
	// proofreader.
	MaxEditDistance float64 `json:"max_edit_distance"`

	// Profiles are the places a text is put right at, by the name a consumer asks for one under.
	Profiles map[string]Profile `json:"profiles"`
}

// Default proofreading sizes.
const (
	DefaultBatchSize = 40
	DefaultInFlight  = 4
)

// Profile is one proofreading profile: how a text is cut into requests. Which
// model answers them is decided by the agent setting of the proofreading plugin.
type Profile struct {
	// BatchSize is how many lines one request carries.
	BatchSize int `json:"batch_size"`
	// Overlap is how many lines neighbouring batches share.
	Overlap int `json:"overlap"`
	// InFlight is how many requests one call answers.
	InFlight int `json:"in_flight,omitempty"`
}

// Proofread names the profile a text is put right at.
type Proofread struct {
	// Profile is the profile, by the name the profiles carry it under.
	Profile string `json:"with"`

	// IsAutomatic is whether a text already written down is put right without anybody asking for it.
	IsAutomatic bool `json:"automatically"`
}

// HasProfiles says whether this installation has any profile configured.
func (c ProofreadingConfig) HasProfiles() bool {
	return len(c.Profiles) > 0
}

// GetInFlight is how many requests this profile has answered at once.
func (p Profile) GetInFlight(planned int) int {
	switch {
	case p.InFlight > 0:
		return p.InFlight
	case planned > 0:
		return planned
	}
	return DefaultInFlight
}

type proofconfigFile struct {
	MaxEditDistance *float64           `json:"max_edit_distance"`
	Profiles        map[string]Profile `json:"profiles"`
}

// UnmarshalJSON keeps defaults for fields omitted from file.
func (c *ProofreadingConfig) UnmarshalJSON(raw []byte) error {
	var f proofconfigFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	*c = ProofreadingConfig{}
	assign(&c.MaxEditDistance, f.MaxEditDistance)
	c.Profiles = f.Profiles
	return nil
}

type profileFile struct {
	BatchSize *int `json:"batch_size"`
	Overlap   *int `json:"overlap"`
	InFlight  *int `json:"in_flight,omitempty"`
}

// UnmarshalJSON starts from the default batch size.
func (p *Profile) UnmarshalJSON(raw []byte) error {
	var f profileFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	*p = Profile{BatchSize: DefaultBatchSize}
	assign(&p.BatchSize, f.BatchSize)
	assign(&p.Overlap, f.Overlap)
	assign(&p.InFlight, f.InFlight)
	return nil
}

// MarshalJSON writes the profile fields.
func (p Profile) MarshalJSON() ([]byte, error) {
	f := profileFile{BatchSize: &p.BatchSize, Overlap: &p.Overlap}
	if p.InFlight > 0 {
		f.InFlight = &p.InFlight
	}
	return json.Marshal(f)
}

func assign[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}

// DefaultIndexing is what an installation nobody has configured makes a vault
// searchable with.
func DefaultIndexing() Indexing {
	set := true
	return Indexing{
		Embedding: DefaultEmbedding(),
		Recognition: Recognition{
			Proofread: Proofread{
				Profile:     "",
				IsAutomatic: false,
			},
		},
		Proofreading: ProofreadingConfig{},
		Transcription: Transcription{
			Proofread: Proofread{
				Profile:     "",
				IsAutomatic: false,
			},
		},
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

// Where an embedding vector is made.
const (
	UseLocal   = "local"
	UseService = "service"
)

// How a model's per-token output becomes one vector.
const (
	PoolMean = "mean"
	PoolHead = "head"
)

// EmbeddingConfig is the embedding section of this installation's settings: what a
// vector is, where it is made, and how near the query a passage stands to be an
// answer at all.
type EmbeddingConfig struct {
	Model    EmbeddingModelConfig `json:"model"`
	Indexing EmbeddingProvider    `json:"indexing"`
	Query    EmbeddingProvider    `json:"query"`
	Floor    float64              `json:"floor"`
}

// EmbeddingModelConfig is what a vector is: everything that decides the space it lands in.
type EmbeddingModelConfig struct {
	Name       string `json:"name"`
	Dimensions int    `json:"dimensions"`
	MaxTokens  int    `json:"max_tokens"`
	Pooling    string `json:"pooling"`
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (m *EmbeddingModelConfig) UnmarshalJSON(raw []byte) error {
	var f struct {
		Name       *string `json:"name"`
		Dimensions *int    `json:"dimensions"`
		MaxTokens  *int    `json:"max_tokens"`
		Pooling    *string `json:"pooling"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&m.Name, f.Name)
	assign(&m.Dimensions, f.Dimensions)
	assign(&m.MaxTokens, f.MaxTokens)
	assign(&m.Pooling, f.Pooling)
	return nil
}

// DefaultEmbedding embeds with empty model and repository defaults (no hardcoded URLs).
func DefaultEmbedding() EmbeddingConfig {
	return EmbeddingConfig{
		Model: EmbeddingModelConfig{
			Name:       "",
			Dimensions: 384,
			MaxTokens:  256,
			Pooling:    PoolMean,
		},
		Indexing: EmbeddingProvider{Use: UseLocal},
		Query:    EmbeddingProvider{},
	}
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (c *EmbeddingConfig) UnmarshalJSON(raw []byte) error {
	var f struct {
		Model    *EmbeddingModelConfig `json:"model"`
		Indexing *EmbeddingProvider    `json:"indexing"`
		Query    *EmbeddingProvider    `json:"query"`
		Floor    *float64              `json:"floor"`
	}
	f.Model, f.Indexing, f.Query = &c.Model, &c.Indexing, &c.Query
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&c.Floor, f.Floor)
	if c.Model.Pooling == "" {
		c.Model.Pooling = PoolMean
	}
	return nil
}

// EmbeddingProvider is where a vector is made: on this machine, or by a service.
type EmbeddingProvider struct {
	Use string `json:"use"`
}
