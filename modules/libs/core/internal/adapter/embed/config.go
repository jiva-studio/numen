package embed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Where a vector is made.
const (
	UseLocal   = "local"
	UseService = "service"
)

// How a model's per-token output becomes one vector. A model pooled the way it
// was not trained to be answers with vectors in a space of its own, near
// nothing the same model made another way.
const (
	PoolMean = "mean"
	PoolHead = "head"
)

// KeyEnvVar is where the service key is read from when the configuration file
// does not carry one.
const KeyEnvVar = "NUMEN_EMBEDDING_KEY"

// The files a model is made of, and where a repository keeps them. ModelFile is
// the build that is run when the configuration names none, and TokenizerFile
// defines the tokeniser in full.
const (
	ModelFile     = "model.onnx"
	TokenizerFile = "tokenizer.json"
	ModelFolder   = "onnx"
)

// Config is the embedding section of this installation's settings: what a
// vector is, where it is made, and how near the query a passage stands to be an
// answer at all.
type Config struct {
	// Model is what a vector is, and it is said once. A vector made while a
	// vault is indexed is claimed again by a question, so the two are one
	// model or the comparison between them means nothing.
	Model Model `json:"model"`

	// Indexing makes the vectors a vault is searched by. Query makes the vector
	// a question is asked with, and taking nothing here is asking the way the
	// vault was indexed.
	//
	// A vault is indexed once and asked all day, and this machine answers a
	// question without a network.
	Indexing Provider `json:"indexing"`
	Query    Provider `json:"query"`

	// Floor is the cosine similarity a passage reaches to be an answer, in the
	// units the model in use measures in. Where a model puts two pieces of text
	// about different things is a fact about that model, so a model changed is
	// a floor measured again. Zero takes the one the search was built against.
	Floor float64 `json:"floor"`
}

// Model is what a vector is: everything that decides the space it lands in.
//
// Name is what the model is called here, and is not how either provider
// reaches it: a repository and a service call one model by two names, and
// vectors made under both are kept under this one.
type Model struct {
	// Name is what the model is called here.
	Name string `json:"name"`
	// Dimensions is how wide its vectors are. The coarse index is built for one
	// width, and changing it builds that index again from the vectors held.
	Dimensions int `json:"dimensions"`
	// MaxTokens is where the model truncates what it is given. A window cut
	// somewhere else is a window whose vector describes text it does not hold.
	MaxTokens int `json:"max_tokens"`
	// Pooling is PoolMean or PoolHead. Empty is PoolMean.
	Pooling string `json:"pooling"`
}

// Stored is this model in the words a vector is kept under, made at the address
// given. It is the one crossing between the settings and the index, so nothing
// copies the fields across by hand.
func (m Model) Stored(from string) port.EmbeddingModel {
	return port.EmbeddingModel{
		Name:       m.Name,
		Dimensions: m.Dimensions,
		MaxTokens:  m.MaxTokens,
		Pooling:    m.Pooling,
		From:       from,
	}
}

// Provider is where a vector is made: on this machine, or by a service.
//
// The settings for both are carried, because a file keeps the one it is not on
// and a person moves between them by changing a word. Which of the two is in
// force is Use, and Local and Service are the only way to the settings behind
// it: the half that is not in force describes nothing this installation does,
// and is not read out of here by mistake.
type Provider struct {
	// Use is UseLocal or UseService. Empty is an installation that makes no
	// vector at all.
	Use string

	local   LocalModel
	service ServiceModel
}

// Local is the model this machine runs, and false where a vector is made
// anywhere else.
func (p Provider) Local() (LocalModel, bool) {
	if p.Use != UseLocal {
		return LocalModel{}, false
	}
	return p.local, true
}

// Service is the service a vector is asked of, and false where a vector is made
// anywhere else.
func (p Provider) Service() (ServiceModel, bool) {
	if p.Use != UseService {
		return ServiceModel{}, false
	}
	return p.service, true
}

// Running is this provider making its vectors on this machine with the model
// given, and Serving is it making them at the service given. Each sets the word
// with the settings, so a half is never written without being put in force, and
// each keeps the half it is not on the way the file does.
func (p Provider) Running(m LocalModel) Provider {
	p.Use, p.local = UseLocal, m
	return p
}

func (p Provider) Serving(m ServiceModel) Provider {
	p.Use, p.service = UseService, m
	return p
}

// From is this provider as the address its vectors are kept under. It leads
// with the word that says which of the two it is, because one name is both a
// repository and something a service answers to.
func (p Provider) From() string {
	if local, ok := p.Local(); ok {
		return local.From()
	}
	if service, ok := p.Service(); ok {
		return service.From()
	}
	return ""
}

// LocalModel is how this machine reaches a model it runs.
type LocalModel struct {
	// Name is a HuggingFace repository.
	Name string `json:"name"`
	// Dir holds the model and tokenizer.json. Empty means the download cache.
	Dir string `json:"dir"`
	// File is the model inside the repository or the directory. Empty is
	// model.onnx, and naming another is how a quantised build of one model is
	// run in place of the full one.
	File string `json:"file"`
	// BatchTexts is how many texts one forward pass carries.
	BatchTexts int `json:"batch_texts"`
	// Download allows fetching the model when it is not on this machine. Turned
	// off, and with no directory named, a vault is searched by its words.
	Download bool `json:"download"`
}

// From is where this machine reads the weights: the directory when one is
// named, and the repository otherwise, with the file that is run inside it. A
// quantised build is a file of its own and answers with numbers of its own.
//
// The file and the directory are settled to one form, so one set of weights has
// one address whichever way the configuration writes it.
func (m LocalModel) From() string {
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

// ServiceModel is how a hosted model is reached over HTTP.
type ServiceModel struct {
	// BaseURL points at anything speaking the /v1/embeddings request shape.
	BaseURL string `json:"base_url"`
	// Name is what that service calls the model.
	Name string `json:"name"`
	// BatchCharacters bounds one request by the characters of everything in it.
	BatchCharacters int `json:"batch_characters"`
	// KeyEnv names the environment variable holding the key, for an
	// installation that keeps it out of the file.
	KeyEnv string `json:"key_env"`

	// key is unexported: no value a caller formats or serialises carries it.
	// Key reads it.
	key string
}

// From is the service and the name it is asked for there. Two services
// answering to one name are two models, and the key is no part of this.
func (s ServiceModel) From() string {
	return UseService + ":" + strings.TrimSuffix(s.BaseURL, "/") + "/" + s.Name
}

// Defaults embed on this machine: no key and no account. The model itself is
// fetched the first time it is wanted.
func Defaults() Config {
	here := Provider{
		local: LocalModel{Name: "intfloat/multilingual-e5-small", BatchTexts: 8, Download: true},
		service: ServiceModel{
			BaseURL:         "https://api.openai.com/v1",
			Name:            "text-embedding-3-small",
			BatchCharacters: 32000,
			KeyEnv:          KeyEnvVar,
		},
	}
	indexing := here
	indexing.Use = UseLocal
	return Config{
		Model: Model{
			Name:       "intfloat/multilingual-e5-small",
			Dimensions: 384,
			MaxTokens:  256,
			Pooling:    PoolMean,
		},
		Indexing: indexing,
		Query:    here,
	}
}

// Asking is where the vector of a question is made. An installation that says
// nothing about questions asks the way it indexed.
func (c Config) Asking() Provider {
	if c.Query.Use == "" {
		return c.Indexing
	}
	return c.Query
}

// Stored is the identity every vector this installation keeps is filed under:
// the model, made where the index is filled. A question is embedded wherever
// the settings place it and claims the rows already there.
func (c Config) Stored() port.EmbeddingModel {
	return c.Model.Stored(c.Indexing.From())
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (c *Config) UnmarshalJSON(raw []byte) error {
	var f struct {
		Model    *Model    `json:"model"`
		Indexing *Provider `json:"indexing"`
		Query    *Provider `json:"query"`
		Floor    *float64  `json:"floor"`
	}
	f.Model, f.Indexing, f.Query = &c.Model, &c.Indexing, &c.Query
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&c.Floor, f.Floor)
	// A model is pooled one way, under one word. Two words for one pooling are
	// two keys over one set of vectors.
	if c.Model.Pooling == "" {
		c.Model.Pooling = PoolMean
	}
	return nil
}

// serviceModelFile is the shape on disk, with the key among the fields a person
// writes.
type serviceModelFile struct {
	BaseURL         *string `json:"base_url"`
	Name            *string `json:"name"`
	BatchCharacters *int    `json:"batch_characters"`
	KeyEnv          *string `json:"key_env"`
	// A key is written by a person and never by us.
	Key *string `json:"key,omitempty"`
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (s *ServiceModel) UnmarshalJSON(raw []byte) error {
	var f serviceModelFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&s.BaseURL, f.BaseURL)
	assign(&s.Name, f.Name)
	assign(&s.BatchCharacters, f.BatchCharacters)
	assign(&s.KeyEnv, f.KeyEnv)
	assign(&s.key, f.Key)
	return nil
}

// MarshalJSON writes everything but the key. Rewriting the file is not how a
// key is set.
func (s ServiceModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(serviceModelFile{
		BaseURL:         &s.BaseURL,
		Name:            &s.Name,
		BatchCharacters: &s.BatchCharacters,
		KeyEnv:          &s.KeyEnv,
	})
}

// String reports the configuration with the key replaced.
func (s ServiceModel) String() string {
	held := "absent"
	if s.Key() != "" {
		held = "present"
	}
	return "service " + s.Name + " at " + s.BaseURL + ", key " + held
}

// Key is the service key: from the file if it names one, otherwise from the
// environment. It is never taken as an argument.
func (s ServiceModel) Key() string {
	if s.key != "" {
		return s.key
	}
	name := s.KeyEnv
	if name == "" {
		name = KeyEnvVar
	}
	return os.Getenv(name)
}

func assign[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (m *LocalModel) UnmarshalJSON(raw []byte) error {
	var f struct {
		Name       *string `json:"name"`
		Dir        *string `json:"dir"`
		File       *string `json:"file"`
		BatchTexts *int    `json:"batch_texts"`
		Download   *bool   `json:"download"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&m.Name, f.Name)
	assign(&m.Dir, f.Dir)
	assign(&m.File, f.File)
	assign(&m.BatchTexts, f.BatchTexts)
	assign(&m.Download, f.Download)
	return nil
}

// providerFile is the shape on disk. The keys are written down here because the
// settings behind them are unexported, and a file anybody already has is read
// and written by these three names whatever the fields come to be called.
type providerFile struct {
	// Use is `local` or `service`, and names which of the two sections below is
	// the one in force. Empty makes no vector at all.
	Use *string `json:"use"`

	Local   *LocalModel   `json:"local"`
	Service *ServiceModel `json:"service"`
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (p *Provider) UnmarshalJSON(raw []byte) error {
	f := providerFile{Local: &p.local, Service: &p.service}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&p.Use, f.Use)
	return nil
}

// MarshalJSON writes both halves, the one in force and the one kept: a person
// moving between them by a word finds the settings they left still there.
func (p Provider) MarshalJSON() ([]byte, error) {
	return json.Marshal(providerFile{Use: &p.Use, Local: &p.local, Service: &p.service})
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (i *Model) UnmarshalJSON(raw []byte) error {
	var f struct {
		Name       *string `json:"name"`
		Dimensions *int    `json:"dimensions"`
		MaxTokens  *int    `json:"max_tokens"`
		Pooling    *string `json:"pooling"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&i.Name, f.Name)
	assign(&i.Dimensions, f.Dimensions)
	assign(&i.MaxTokens, f.MaxTokens)
	assign(&i.Pooling, f.Pooling)
	return nil
}
