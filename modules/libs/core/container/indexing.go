package container

import (
	"encoding/json"
	"math"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
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
	RecognitionConfig

	// Proofread names the profile a recognition is put right at. Automatically
	// there says whether a recognition just made is put right without anybody
	// asking.
	Proofread Proofread `json:"proofread"`
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
	TranscriptionConfig

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
// model answers them is decided by the route of the proofreading purpose.
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

// TranscriptionConfig holds the configuration for audio and video transcription.
type TranscriptionConfig struct {
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

	Model     ParakeetModel  `json:"model"`
	Segmenter SegmenterModel `json:"segmenter"`
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

// SegmenterModel finds where in a recording somebody is speaking. What it cuts is
// what the transcriber is given, so where it cuts is part of what the words
// are.
type SegmenterModel struct {
	// Name is what this segmenter is called in the record kept beside a text.
	Name string `json:"name"`
	// Repo is where the model is fetched from, and Path is a file on this
	// machine. A path is used as given; Repo is looked for in Dir first.
	Repo string `json:"from"`
	Path string `json:"path"`

	// Threshold is how sure the model has to be that a window carries speech.
	Threshold float32 `json:"threshold"`
	// Silence is how much quiet, in milliseconds, closes a segment.
	Silence int `json:"silence"`
	// Pad is how many milliseconds are kept on each side of a segment, so that
	// the first and last sound of a word are inside it.
	Pad int `json:"pad"`
	// Longest is how many milliseconds one segment may run to. Speech that goes
	// on longer is cut at the quietest window this side of the limit.
	Longest int `json:"longest"`
	// Shortest is how many milliseconds a segment carries to be a segment at
	// all.
	Shortest int `json:"shortest"`
	// Least is how many milliseconds a segment runs to before it stands as a
	// line of its own. A shorter one is put together with the segment after it.
	Least int `json:"least"`
}

// DefaultIndexing is what an installation nobody has configured makes a vault
// searchable with.
func DefaultIndexing() Indexing {
	set := true
	return Indexing{
		Embedding: DefaultEmbedding(),
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
			Proofread: Proofread{
				Profile:     "",
				IsAutomatic: false,
			},
		},
		Proofreading: ProofreadingConfig{},
		Transcription: Transcription{
			TranscriptionConfig: TranscriptionConfig{
				Runtime:        "",
				Dir:            "",
				ShouldDownload: true,
				Threads:        0,
				Model: ParakeetModel{
					Name: "",
					Repo: "",
				},
				Segmenter: SegmenterModel{
					Name: "",
					Repo: "",
				},
			},
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

// What runs a model on this machine.
const (
	EngineRuntime = "onnxruntime"
	EnginePureGo  = "go"
)

// The files a model is made of.
const (
	ModelFile     = "model.onnx"
	TokenizerFile = "tokenizer.json"
	ModelFolder   = "onnx"
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

// GetStoredModel is this model in the words a vector is kept under, made at the
// address given.
func (m EmbeddingModelConfig) GetStoredModel(from string) port.EmbeddingModel {
	return port.EmbeddingModel{
		Name:       m.Name,
		Dimensions: m.Dimensions,
		MaxTokens:  m.MaxTokens,
		Pooling:    m.Pooling,
		From:       from,
	}
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
	here := EmbeddingProvider{
		local: LocalModel{
			Name:           "",
			BatchTexts:     32,
			ShouldDownload: true,
		},
		service: ServiceModel{
			BaseURL:         "",
			Name:            "",
			BatchCharacters: 32000,
		},
	}
	indexing := here
	indexing.Use = UseLocal
	return EmbeddingConfig{
		Model: EmbeddingModelConfig{
			Name:       "",
			Dimensions: 384,
			MaxTokens:  256,
			Pooling:    PoolMean,
		},
		Indexing: indexing,
		Query:    here,
	}
}

// GetQueryProvider is where the vector of a question is made.
func (c EmbeddingConfig) GetQueryProvider() EmbeddingProvider {
	if c.Query.Use == "" {
		return c.Indexing
	}
	return c.Query
}

// GetStoredModel is the identity every vector this installation keeps is filed
// under.
func (c EmbeddingConfig) GetStoredModel() port.EmbeddingModel {
	return c.Model.GetStoredModel(c.Indexing.GetAddress())
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

	local   LocalModel
	service ServiceModel
}

func (p EmbeddingProvider) Local() (LocalModel, bool) {
	if p.Use != UseLocal {
		return LocalModel{}, false
	}
	return p.local, true
}

func (p EmbeddingProvider) Service() (ServiceModel, bool) {
	if p.Use != UseService {
		return ServiceModel{}, false
	}
	return p.service, true
}

func (p EmbeddingProvider) SetLocal(m LocalModel) EmbeddingProvider {
	p.Use, p.local = UseLocal, m
	return p
}

func (p EmbeddingProvider) SetService(m ServiceModel) EmbeddingProvider {
	p.Use, p.service = UseService, m
	return p
}

func (p EmbeddingProvider) GetAddress() string {
	if local, ok := p.Local(); ok {
		return local.GetAddress()
	}
	if service, ok := p.Service(); ok {
		return service.GetAddress()
	}
	return ""
}

type providerFile struct {
	Use     *string       `json:"use"`
	Local   *LocalModel   `json:"local"`
	Service *ServiceModel `json:"service"`
}

func (p *EmbeddingProvider) UnmarshalJSON(raw []byte) error {
	f := providerFile{Local: &p.local, Service: &p.service}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&p.Use, f.Use)
	return nil
}

func (p EmbeddingProvider) MarshalJSON() ([]byte, error) {
	return json.Marshal(providerFile{Use: &p.Use, Local: &p.local, Service: &p.service})
}

// LocalModel is how this machine reaches a model it runs.
type LocalModel struct {
	Name           string `json:"name"`
	Dir            string `json:"dir"`
	File           string `json:"file"`
	BatchTexts     int    `json:"batch_texts"`
	Engine         string `json:"engine"`
	Runtime        string `json:"runtime"`
	Threads        int    `json:"threads"`
	Sessions       int    `json:"sessions"`
	Provider       string `json:"provider"`
	ShouldDownload bool   `json:"download"`
}

func (m LocalModel) GetAddress() string {
	at := m.Name
	if m.Dir != "" {
		at = filepath.ToSlash(filepath.Clean(m.Dir))
	}
	file := m.File
	if file == "" {
		file = ModelFile
	}
	return UseLocal + ":" + at + "/" + file
}

func (m *LocalModel) UnmarshalJSON(raw []byte) error {
	var f struct {
		Name       *string `json:"name"`
		Dir        *string `json:"dir"`
		File       *string `json:"file"`
		BatchTexts *int    `json:"batch_texts"`
		BatchSize  *int    `json:"batch_size"`
		Engine     *string `json:"engine"`
		Runtime    *string `json:"runtime"`
		Threads    *int    `json:"threads"`
		Sessions   *int    `json:"sessions"`
		Provider   *string `json:"provider"`
		Download   *bool   `json:"download"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&m.Name, f.Name)
	assign(&m.Dir, f.Dir)
	assign(&m.File, f.File)
	if f.BatchSize != nil {
		assign(&m.BatchTexts, f.BatchSize)
	} else {
		assign(&m.BatchTexts, f.BatchTexts)
	}
	assign(&m.Engine, f.Engine)
	assign(&m.Runtime, f.Runtime)
	assign(&m.Threads, f.Threads)
	assign(&m.Sessions, f.Sessions)
	assign(&m.Provider, f.Provider)
	assign(&m.ShouldDownload, f.Download)
	return nil
}

// ServiceModel is how a hosted model is reached over HTTP.
type ServiceModel struct {
	BaseURL         string `json:"base_url"`
	Name            string `json:"name"`
	BatchCharacters int    `json:"batch_characters"`
}

func (s ServiceModel) GetAddress() string {
	return UseService + ":" + strings.TrimSuffix(s.BaseURL, "/") + "/" + s.Name
}

type serviceModelFile struct {
	BaseURL         *string `json:"base_url"`
	Name            *string `json:"name"`
	BatchCharacters *int    `json:"batch_characters"`
}

func (s *ServiceModel) UnmarshalJSON(raw []byte) error {
	var f serviceModelFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&s.BaseURL, f.BaseURL)
	assign(&s.Name, f.Name)
	assign(&s.BatchCharacters, f.BatchCharacters)
	return nil
}

func (s ServiceModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(serviceModelFile{
		BaseURL:         &s.BaseURL,
		Name:            &s.Name,
		BatchCharacters: &s.BatchCharacters,
	})
}
