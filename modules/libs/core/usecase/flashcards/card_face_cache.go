package flashcards

import (
	"slices"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// maxCachedDecks is the most decks a window keeps parsed. Past it the cache is
// emptied and refilled by the requests that follow.
const maxCachedDecks = 4096

// deckParsers is the most decks parsed at once.
const deckParsers = 4

type deckKey struct {
	vault domain.VaultID
	path  string
}

// deckFaces is the card faces one deck held when it was parsed, and the files
// they were laid out from. It is read-only once stored.
type deckFaces struct {
	deck     domain.Fingerprint
	stencils map[string]domain.Fingerprint
	faces    []CardFace
}

// isCurrent says whether the deck and every stencil it was cut by are the files
// the faces were parsed from.
func (e deckFaces) isCurrent(deck domain.Fingerprint, stencils map[string]domain.Fingerprint) bool {
	return isSameFile(e.deck, deck) && len(e.stencils) == len(stencils) &&
		mapsEqualBy(e.stencils, stencils)
}

func isSameFile(a, b domain.Fingerprint) bool {
	return a.Size == b.Size && a.ModTime.Equal(b.ModTime)
}

func mapsEqualBy(a, b map[string]domain.Fingerprint) bool {
	for path, one := range a {
		other, held := b[path]
		if !held || !isSameFile(one, other) {
			return false
		}
	}
	return true
}

// faceCache is the parsed faces of the decks a window has read, for the life of
// the window. A nil cache holds nothing.
type faceCache struct {
	mu    sync.Mutex
	decks map[deckKey]deckFaces
	// epoch counts invalidations, and a parse that began before one is not
	// stored.
	epoch uint64
}

func newFaceCache() *faceCache {
	return &faceCache{decks: make(map[deckKey]deckFaces)}
}

// begin returns the epoch a parse starts in.
func (c *faceCache) begin() uint64 {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.epoch
}

func (c *faceCache) get(key deckKey) (deckFaces, bool) {
	if c == nil {
		return deckFaces{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, held := c.decks[key]
	return e, held
}

// put stores a deck parsed in the given epoch.
func (c *faceCache) put(epoch uint64, key deckKey, e deckFaces) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if epoch != c.epoch {
		return
	}
	if _, held := c.decks[key]; !held && len(c.decks) >= maxCachedDecks {
		clear(c.decks)
	}
	c.decks[key] = e
}

func (c *faceCache) drop(key deckKey) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.decks, key)
}

// keep drops every entry of the vault whose path is not in paths.
func (c *faceCache) keep(vault domain.VaultID, paths []string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.decks {
		if key.vault == vault && !slices.Contains(paths, key.path) {
			delete(c.decks, key)
		}
	}
}

// invalidate drops the decks at paths and every deck cut by a stencil at one of
// them.
func (c *faceCache) invalidate(vault domain.VaultID, paths []string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.epoch++
	for key, e := range c.decks {
		if key.vault != vault {
			continue
		}
		if slices.Contains(paths, key.path) || usesAny(e.stencils, paths) {
			delete(c.decks, key)
		}
	}
}

func usesAny(stencils map[string]domain.Fingerprint, paths []string) bool {
	for _, path := range paths {
		if _, held := stencils[path]; held {
			return true
		}
	}
	return false
}

func (c *faceCache) invalidateAll() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.epoch++
	clear(c.decks)
}
