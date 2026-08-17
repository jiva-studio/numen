package embed

import (
	"encoding/json"
	"os"
)

// Which embedder an installation uses.
const (
	UseLocal   = "local"
	UseService = "service"
)

// KeyEnvVar is where the service key is read from when the configuration file
// does not carry one.
const KeyEnvVar = "NUMEN_EMBEDDING_KEY"

// Config is the embedding section of this installation's settings: which
// embedder, which model, how wide its vectors are.
type Config struct {
	Use     string       `json:"use"`
	Local   LocalModel   `json:"local"`
	Service ServiceModel `json:"service"`
}

// LocalModel is a model this machine runs.
type LocalModel struct {
	// Name is a HuggingFace repository, which is also the identity stored with
	// every vector it produces.
	Name string `json:"name"`
	// Dimensions is what the model returns. A model of another width is refused.
	Dimensions int `json:"dimensions"`
	// Dir holds model.onnx and tokenizer.json. Empty means the download cache.
	Dir string `json:"dir"`
	// MaxTokens is where a text is truncated. It stays under the model's own
	// limit, since a silently truncated window indexes text it does not contain.
	MaxTokens int `json:"max_tokens"`
	// BatchTexts is how many texts one forward pass carries.
	BatchTexts int `json:"batch_texts"`
	// Download allows fetching the model when it is not on this machine, and is
	// off by default. Without it and without a directory, a vault is searched by
	// its words.
	Download bool `json:"download"`
}

// ServiceModel is a hosted model reached over HTTP.
type ServiceModel struct {
	BaseURL    string `json:"base_url"`
	Name       string `json:"name"`
	Dimensions int    `json:"dimensions"`
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
	return Config{
		Use: UseLocal,
		Local: LocalModel{
			Name:       "intfloat/multilingual-e5-small",
			Dimensions: 384,
			MaxTokens:  256,
			BatchTexts: 8,
		},
		Service: ServiceModel{
			BaseURL:         "https://api.openai.com/v1",
			Name:            "text-embedding-3-small",
			Dimensions:      1536,
			BatchCharacters: 32000,
			KeyEnv:          KeyEnvVar,
		},
	}
}

// serviceFile is the shape on disk, with the key among the fields a person
// writes.
type serviceFile struct {
	BaseURL         *string `json:"base_url"`
	Name            *string `json:"name"`
	Dimensions      *int    `json:"dimensions"`
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
	assign(&s.Dimensions, f.Dimensions)
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
		Dimensions:      &s.Dimensions,
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
		Dimensions *int    `json:"dimensions"`
		Dir        *string `json:"dir"`
		MaxTokens  *int    `json:"max_tokens"`
		BatchTexts *int    `json:"batch_texts"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&m.Name, f.Name)
	assign(&m.Dimensions, f.Dimensions)
	assign(&m.Dir, f.Dir)
	assign(&m.MaxTokens, f.MaxTokens)
	assign(&m.BatchTexts, f.BatchTexts)
	return nil
}
