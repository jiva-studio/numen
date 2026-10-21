package flashcards

import (
	"container/list"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// defaultCachedFaces is how many card faces a window keeps parsed when
// ListCardFaces.MaxFaces names none. Past it the least recently used decks are
// forgotten one at a time.
const defaultCachedFaces = 1 << 18

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
	if !e.deck.IsUnchanged(deck) || len(e.stencils) != len(stencils) {
		return false
	}
	for path, one := range e.stencils {
		other, target := stencils[path]
		if !target || !one.IsUnchanged(other) {
			return false
		}
	}
	return true
}

type cached struct {
	key deckKey
	deckFaces
}

// faceCache is the parsed faces of the decks a window has read, for the life of
// the window. A nil cache holds nothing.
type faceCache struct {
	mu    sync.Mutex
	decks map[deckKey]*list.Element
	// order runs from the most recently used deck to the least.
	order *list.List
	// total is the number of card faces held.
	total int
	// epoch counts invalidations, and a parse that began before one is not
	// stored.
	epoch uint64
}

func newFaceCache() *faceCache {
	return &faceCache{decks: make(map[deckKey]*list.Element), order: list.New()}
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
	at, target := c.decks[key]
	if !target {
		return deckFaces{}, false
	}
	c.order.MoveToFront(at)
	return getEntry(at).deckFaces, true
}

// put stores a deck parsed in the given epoch, then forgets the least recently
// used decks until no more than limit faces are held. A deck holding more than
// limit faces is not stored.
func (c *faceCache) put(epoch uint64, key deckKey, e deckFaces, limit int) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if epoch != c.epoch || len(e.faces) > limit {
		return
	}
	c.remove(key)
	c.decks[key] = c.order.PushFront(&cached{key: key, deckFaces: e})
	c.total += len(e.faces)
	for c.total > limit {
		c.removeElement(c.order.Back())
	}
}

// remove forgets the deck at key. The caller holds the lock.
func (c *faceCache) remove(key deckKey) {
	if at, target := c.decks[key]; target {
		c.removeElement(at)
	}
}

func (c *faceCache) removeElement(at *list.Element) {
	one := getEntry(at)
	c.order.Remove(at)
	delete(c.decks, one.key)
	c.total -= len(one.faces)
}

func (c *faceCache) drop(key deckKey) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.remove(key)
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
			c.remove(key)
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
	for key, at := range c.decks {
		if key.vault != vault {
			continue
		}
		if slices.Contains(paths, key.path) || usesAny(getEntry(at).stencils, paths) {
			c.removeElement(at)
		}
	}
}

func usesAny(stencils map[string]domain.Fingerprint, paths []string) bool {
	for _, path := range paths {
		if _, target := stencils[path]; target {
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
	c.order.Init()
	c.total = 0
}

// getSize returns the number of card faces held.
func (c *faceCache) getSize() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total
}

// hasDeck says whether the deck at key is held.
func (c *faceCache) hasDeck(key deckKey) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, target := c.decks[key]
	return target
}

// getEntry returns the deck an element of the order stands for. Only decks are put in
// it, so the zero entry is never seen.
func getEntry(at *list.Element) *cached {
	one, _ := at.Value.(*cached)
	if one == nil {
		return &cached{}
	}
	return one
}
