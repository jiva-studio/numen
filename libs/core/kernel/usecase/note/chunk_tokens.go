package note

import (
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

// ChunkTokens is the chunk size in tokens that the search source reports, shared by every copy of the
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

// Follow holds the chunk size of the search source, and again each time onChange has its description change.
func (c *ChunkTokens) Follow(source searchsourceport.Source, onChange func(func())) {
	c.Set(source.Describe().ChunkTokens)
	onChange(func() { c.Set(source.Describe().ChunkTokens) })
}
