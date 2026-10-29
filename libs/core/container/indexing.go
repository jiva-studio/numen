package container

import "github.com/jiva-studio/numen/libs/core/adapter/settings"

// Indexing is how a vault is made searchable.
type Indexing = settings.Indexing

// DefaultIndexing is what an unconfigured installation uses to make a vault searchable.
func DefaultIndexing() Indexing { return settings.DefaultIndexing() }

// EmbeddingConfig is the embedding section of this installation's settings.
type EmbeddingConfig = settings.EmbeddingConfig

// DefaultEmbedding cuts chunks a model reading 256 tokens takes whole.
func DefaultEmbedding() EmbeddingConfig { return settings.DefaultEmbedding() }
