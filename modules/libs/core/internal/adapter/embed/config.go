package embed

import (
	"encoding/json"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// How a model's per-token output becomes one vector. A model pooled the way it
// was not trained to be answers with vectors in a space of its own, near
// nothing the same model made another way.
const (
	PoolMean = "mean"
	PoolHead = "head"
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

// GetStoredModel is this model in the words a vector is kept under, made at the
// address given. It is the one crossing between the settings and the index, so
// nothing copies the fields across by hand.
func (m Model) GetStoredModel(from string) port.EmbeddingModel {
	return port.EmbeddingModel{
		Name:       m.Name,
		Dimensions: m.Dimensions,
		MaxTokens:  m.MaxTokens,
		Pooling:    m.Pooling,
		From:       from,
	}
}

// Defaults embed on this machine: no key and no account. The model itself is
// fetched the first time it is wanted.
func Defaults() Config {
	here := Provider{
		local: LocalModel{Name: "intfloat/multilingual-e5-small", BatchTexts: 8, ShouldDownload: true},
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

// GetQueryProvider is where the vector of a question is made. An installation
// that says nothing about questions asks the way it indexed.
func (c Config) GetQueryProvider() Provider {
	if c.Query.Use == "" {
		return c.Indexing
	}
	return c.Query
}

// GetStoredModel is the identity every vector this installation keeps is filed
// under: the model, made where the index is filled. A question is embedded
// wherever the settings place it and claims the rows already there.
func (c Config) GetStoredModel() port.EmbeddingModel {
	return c.Model.GetStoredModel(c.Indexing.GetAddress())
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

func assign[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (m *Model) UnmarshalJSON(raw []byte) error {
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
