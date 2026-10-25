package container

import "encoding/json"

// Indexing is how a vault is made searchable. It is one section of the settings
// file and the union of what the adapters are configured with, which is why it
// stands where every adapter is bound.
type Indexing struct {
	// Embedding is which model turns text into vectors, and how it is reached.
	Embedding EmbeddingConfig `json:"embedding"`

	// Proofreading is what puts a transcript right. Naming no profile here is
	// naming no proofreader, and a transcript is used as it was heard.
	Proofreading ProofreadingConfig `json:"proofreading"`
}

// ProofreadingConfig is the proofreading section of indexing settings.
type ProofreadingConfig struct {
	Profiles map[string]any `json:"profiles"`
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

func assign[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}

// DefaultIndexing is what an installation nobody has configured makes a vault
// searchable with.
func DefaultIndexing() Indexing {
	return Indexing{
		Embedding: DefaultEmbedding(),
		Proofreading: ProofreadingConfig{
			Profiles: map[string]any{},
		},
	}
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
