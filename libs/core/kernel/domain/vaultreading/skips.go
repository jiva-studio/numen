package vaultreading

import "sync"

// Skips is the sources whose indexing a person canceled. It is held in memory only, so a restart indexes
// them again.
//
// A source is skipped by the vector pass while its content hash is the one the pass first saw after the
// cancel, and by the text pass until it changes. A change the vault reports for its path clears it from
// both.
type Skips struct {
	mu         sync.Mutex
	vectors    map[string]*string
	texts      map[string]bool
	chunks     int64
	generation uint64
}

// NewSkips returns skips that leave nothing out.
func NewSkips() *Skips {
	return &Skips{vectors: map[string]*string{}, texts: map[string]bool{}}
}

func (c *Skips) AddVectors(path string) {
	if path == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vectors[path] = nil
	c.forgetChunks()
}

func (c *Skips) AddTexts(path string) {
	if path == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.texts[path] = true
}

// IsVectorSkipped reports whether the vectors of the source at the path are skipped for the content with
// the hash. The first call after the cancel records the hash.
func (c *Skips) IsVectorSkipped(path, hash string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	seen, isListed := c.vectors[path]
	switch {
	case !isListed:
		return false
	case seen == nil:
		c.vectors[path] = &hash
		return true
	case *seen == hash:
		return true
	default:
		delete(c.vectors, path)
		c.forgetChunks()
		return false
	}
}

// IsTextSkipped reports whether the text of the source at the path is skipped.
func (c *Skips) IsTextSkipped(path string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.texts[path]
}

// Clear stops skipping the sources at the paths.
func (c *Skips) Clear(paths []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, path := range paths {
		delete(c.vectors, path)
		delete(c.texts, path)
	}
	c.forgetChunks()
}

func (c *Skips) forgetChunks() {
	c.chunks = 0
	c.generation++
}

// GetGeneration is how many times the skipped chunks were forgotten.
func (c *Skips) GetGeneration() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

// SetSkippedChunks records how many chunks were skipped, unless the set changed since the generation.
func (c *Skips) SetSkippedChunks(n int, generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if generation == c.generation {
		c.chunks = int64(n)
	}
}

// GetSkippedChunks is how many chunks the skipped sources hold.
func (c *Skips) GetSkippedChunks() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.chunks
}
