package indexing

import "sync"

// skipSet is the sources whose indexing a person canceled. It is
// held in memory only, so a restart indexes them again.
//
// A source is skipped by the vector pass while its content hash is the one the
// pass first saw after the cancel, and by the text pass until it changes. A
// change the vault reports for its path clears it from both.
type skipSet struct {
	mu         sync.Mutex
	vectors    map[string]*string
	texts      map[string]bool
	chunks     int64
	generation uint64
}

func newSkipSet() *skipSet {
	return &skipSet{vectors: map[string]*string{}, texts: map[string]bool{}}
}

func (c *skipSet) addVectors(path string) {
	if path == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vectors[path] = nil
	c.forgetChunks()
}

func (c *skipSet) addTexts(path string) {
	if path == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.texts[path] = true
}

func (c *skipSet) isVectorSkipped(path, hash string) bool {
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

func (c *skipSet) isTextSkipped(path string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.texts[path]
}

func (c *skipSet) clear(paths []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, path := range paths {
		delete(c.vectors, path)
		delete(c.texts, path)
	}
	c.forgetChunks()
}

func (c *skipSet) forgetChunks() {
	c.chunks = 0
	c.generation++
}

func (c *skipSet) getGeneration() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

func (c *skipSet) setSkippedChunks(n int, generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if generation == c.generation {
		c.chunks = int64(n)
	}
}

func (c *skipSet) getSkippedChunks() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.chunks
}

// skipping tells which sources a person canceled the indexing of.
type skipping interface {
	addVectors(path string)
	addTexts(path string)
	isVectorSkipped(path, hash string) bool
	isTextSkipped(path string) bool
	clear(paths []string)
	getGeneration() uint64
	setSkippedChunks(n int, generation uint64)
	getSkippedChunks() int64
}
