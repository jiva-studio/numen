package container

// Indexing is how a vault is made searchable. It is one section of the settings
// file and the union of what the adapters are configured with, which is why it
// stands where every adapter is bound.
type Indexing struct {
	// Embedding is how the index keeps and asks vectors.
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

// EmbeddingConfig is the embedding section of this installation's settings: how
// the index keeps and asks the vectors the embedding plugin makes. Which model
// makes them is the plugin's own setting.
type EmbeddingConfig struct {
	// MaxTokens is the input bound, in tokens, of a chunk the index keeps a
	// vector of. Zero cuts at the chunking defaults.
	MaxTokens int `json:"max_tokens"`
	// Floor is how near a question a passage stands to be an answer at all.
	Floor float64 `json:"floor"`
}

// DefaultEmbedding cuts chunks a model reading 256 tokens takes whole.
func DefaultEmbedding() EmbeddingConfig {
	return EmbeddingConfig{MaxTokens: 256}
}
