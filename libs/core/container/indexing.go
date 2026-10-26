package container

// Indexing is how a vault is made searchable. It is one section of the settings
// file and the union of what the adapters are configured with, which is why it
// stands where every adapter is bound.
type Indexing struct {
	// Embedding is how the index keeps and asks vectors.
	Embedding EmbeddingConfig `json:"embedding"`
}

// DefaultIndexing is what an installation nobody has configured makes a vault
// searchable with.
func DefaultIndexing() Indexing {
	return Indexing{
		Embedding: DefaultEmbedding(),
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
