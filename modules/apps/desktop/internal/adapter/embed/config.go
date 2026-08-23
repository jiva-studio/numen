package embed

import (
	"encoding/json"
	"os"
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

// Config is the embedding section of this installation's settings: what a
// vector is, where it is made, and how near the query a passage stands to be an
// answer at all.
type Config struct {
	// Model is what a vector is, and it is said once. A vector made while a
	// vault is indexed is claimed again by a question, so the two are one
	// model or the comparison between them means nothing.
	Model Identity `json:"model"`

	// Indexing makes the vectors a vault is searched by. Query makes the vector
	// a question is asked with, and taking nothing here is asking the way the
	// vault was indexed.
	//
	// They are separate because their costs are opposite. A vault is indexed
	// once, and a service does that in an hour where this machine takes a day;
	// a question is asked all day, and a machine answers it without a network.
	Indexing Placement `json:"indexing"`
	Query    Placement `json:"query"`

	// Floor is the cosine similarity a passage reaches to be an answer, in the
	// units the model in use measures in. Where a model puts two pieces of text
	// about different things is a fact about that model, so a model changed is
	// a floor measured again. Zero takes the one the search was built against.
	Floor float64 `json:"floor"`
}

// Identity is what a vector is: everything that decides the space it lands in.
//
// Name is what the model is called here, and is not how either placement
// reaches it: a repository and a service call one model by two names, and
// vectors made under both are kept under this one.
type Identity struct {
	Name       string `json:"name"`
	Dimensions int    `json:"dimensions"`
	// MaxTokens is where the model truncates what it is given. A window cut
	// somewhere else is a window whose vector describes text it does not hold.
	MaxTokens int `json:"max_tokens"`
	// Pooling is PoolMean or PoolHead. Empty is PoolMean.
	Pooling string `json:"pooling"`
}

// Placement is where a vector is made: on this machine, or by a service.
type Placement struct {
	Use     string       `json:"use"`
	Local   LocalModel   `json:"local"`
	Service ServiceModel `json:"service"`
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
	// Download allows fetching the model when it is not on this machine, and is
	// off by default. Without it and without a directory, a vault is searched by
	// its words.
	Download bool `json:"download"`
}

// ServiceModel is how a hosted model is reached over HTTP.
type ServiceModel struct {
	BaseURL string `json:"base_url"`
	Name    string `json:"name"`
	// BatchCharacters bounds one request by the characters of everything in it.
	BatchCharacters int `json:"batch_characters"`
	// KeyEnv names the environment variable holding the key, for an
	// installation that keeps it out of the file.
	KeyEnv string `json:"key_env"`

	// key is unexported: no value a caller formats or serialises carries it.
	// Key reads it.
	key string
}

// Defaults embed locally: no key, no account, nothing to reach over a network.
func Defaults() Config {
	here := Placement{
		Local: LocalModel{Name: "intfloat/multilingual-e5-small", BatchTexts: 8},
		Service: ServiceModel{
			BaseURL:         "https://api.openai.com/v1",
			Name:            "text-embedding-3-small",
			BatchCharacters: 32000,
			KeyEnv:          KeyEnvVar,
		},
	}
	indexing := here
	indexing.Use = UseLocal
	return Config{
		Model: Identity{
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
func (c Config) Asking() Placement {
	if c.Query.Use == "" {
		return c.Indexing
	}
	return c.Query
}

// As is a configuration in which this placement is the one that makes every
// vector. A run that asks questions and fills no index opens the placement that
// answers them and no other.
func (p Placement) As(is Identity) Config {
	return Config{Model: is, Indexing: p}
}

// settingsFile is the shape on disk. Both the sections and the older flat form
// are read: an installation configured before questions had a placement of
// their own keeps working, and indexes and asks the one way it named.
type settingsFile struct {
	Model    *json.RawMessage `json:"model"`
	Indexing *json.RawMessage `json:"indexing"`
	Query    *json.RawMessage `json:"query"`
	Floor    *float64         `json:"floor"`

	Use     *string          `json:"use"`
	Local   *json.RawMessage `json:"local"`
	Service *json.RawMessage `json:"service"`
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (c *Config) UnmarshalJSON(raw []byte) error {
	var f settingsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&c.Floor, f.Floor)

	if f.Use != nil || f.Local != nil || f.Service != nil {
		assign(&c.Indexing.Use, f.Use)
		if err := lift(&c.Indexing, &c.Model, f.Local, f.Service); err != nil {
			return err
		}
		c.Query = c.Indexing
		c.Query.Use = ""
	}

	for _, onto := range []struct {
		said *json.RawMessage
		into any
	}{{f.Model, &c.Model}, {f.Indexing, &c.Indexing}, {f.Query, &c.Query}} {
		if onto.said == nil {
			continue
		}
		if err := json.Unmarshal(*onto.said, onto.into); err != nil {
			return err
		}
	}
	return nil
}

// lift reads the flat form: one placement, with what the model is written
// among the two halves of it.
func lift(one *Placement, model *Identity, local, service *json.RawMessage) error {
	var said struct {
		Name       *string `json:"name"`
		Dimensions *int    `json:"dimensions"`
		MaxTokens  *int    `json:"max_tokens"`
		Pooling    *string `json:"pooling"`
	}
	from := local
	if one.Use == UseService {
		from = service
	}
	if local != nil {
		if err := json.Unmarshal(*local, &one.Local); err != nil {
			return err
		}
	}
	if service != nil {
		if err := json.Unmarshal(*service, &one.Service); err != nil {
			return err
		}
	}
	if from == nil {
		return nil
	}
	if err := json.Unmarshal(*from, &said); err != nil {
		return err
	}
	assign(&model.Name, said.Name)
	assign(&model.Dimensions, said.Dimensions)
	assign(&model.MaxTokens, said.MaxTokens)
	assign(&model.Pooling, said.Pooling)
	return nil
}

// serviceFile is the shape on disk, with the key among the fields a person
// writes.
type serviceFile struct {
	BaseURL         *string `json:"base_url"`
	Name            *string `json:"name"`
	BatchCharacters *int    `json:"batch_characters"`
	KeyEnv          *string `json:"key_env"`
	Key             *string `json:"key"`
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (s *ServiceModel) UnmarshalJSON(raw []byte) error {
	var f serviceFile
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
	return json.Marshal(serviceFile{
		BaseURL:         &s.BaseURL,
		Name:            &s.Name,
		BatchCharacters: &s.BatchCharacters,
		KeyEnv:          &s.KeyEnv,
	})
}

// String reports the configuration with the key replaced. A value that formats
// itself cannot be logged into a file by accident.
func (s ServiceModel) String() string {
	held := "absent"
	if s.Key() != "" {
		held = "present"
	}
	return "service " + s.Name + " at " + s.BaseURL + ", key " + held
}

// Key is the service key: from the file if it names one, otherwise from the
// environment. It is never taken as an argument, so there is no call site at
// which it could be written down.
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

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (p *Placement) UnmarshalJSON(raw []byte) error {
	var f struct {
		Use     *string       `json:"use"`
		Local   *LocalModel   `json:"local"`
		Service *ServiceModel `json:"service"`
	}
	f.Local, f.Service = &p.Local, &p.Service
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&p.Use, f.Use)
	return nil
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (i *Identity) UnmarshalJSON(raw []byte) error {
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
