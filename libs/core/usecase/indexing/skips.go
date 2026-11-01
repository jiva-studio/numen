package indexing

import "sync"

// skipSet is the sources whose indexing a person cancelled. It is
// held in memory only, so a restart indexes them again.
//
// A source is skipped by the vector pass while its content hash is the one the
// pass first saw after the cancel, and by the text pass until it changes. A
// change the vault reports for its path clears it from both.
type skipSet struct {
	mu sync.Mutex
	// vectors maps a path to the content hash its chunks carry. A nil hash
	// is a cancel no pass has seen the source under since.
	vectors map[string]*string
	texts   map[string]bool
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
}

func (c *skipSet) addTexts(path string) {
	if path == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.texts[path] = true
}

// isVectorSkipped reports whether the chunks of the source at path, read with
// this content hash, are left without vectors.
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
		return false
	}
}

// isTextSkipped reports whether the source at path is left uncut.
func (c *skipSet) isTextSkipped(path string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.texts[path]
}

// clear forgets the cancel of every path given.
func (c *skipSet) clear(paths []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, path := range paths {
		delete(c.vectors, path)
		delete(c.texts, path)
	}
}
