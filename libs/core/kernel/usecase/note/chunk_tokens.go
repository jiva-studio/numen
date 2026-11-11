package note

import (
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
)

// ChunkTokens is the chunk size in tokens that the embedding plugin reports, shared by every copy of the
// configuration that holds it. A nil ChunkTokens is a size that is not known.
type ChunkTokens struct{ tokens atomic.Int64 }

// Set holds the chunk size in tokens; zero is a size that is not known.
func (c *ChunkTokens) Set(tokens int) {
	if c != nil {
		c.tokens.Store(int64(tokens))
	}
}

// Get is the chunk size in tokens, and zero where it is not known.
func (c *ChunkTokens) Get() int {
	if c == nil {
		return 0
	}
	return int(c.tokens.Load())
}

// Follow holds the chunk size of the model of the embedder, and again each time onChange has the model change.
func (c *ChunkTokens) Follow(embedder embeddingport.Embedder, onChange func(func())) {
	c.Set(embedder.GetModel().ChunkTokens)
	onChange(func() { c.Set(embedder.GetModel().ChunkTokens) })
}
