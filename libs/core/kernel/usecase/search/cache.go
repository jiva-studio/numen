package search

import (
	"sync"
)

// defaultCacheCapacity is how many query embeddings the cache holds.
const defaultCacheCapacity = 256

type cacheKey struct {
	recipe string
	query  string
}

type cacheEntry struct {
	prev   *cacheEntry
	next   *cacheEntry
	key    cacheKey
	vector []float32
}

// QueryCache holds query embeddings in memory.
type QueryCache struct {
	mu       sync.Mutex
	capacity int
	entries  map[cacheKey]*cacheEntry
	head     *cacheEntry
	tail     *cacheEntry
}

// NewQueryCache creates an LRU cache with the capacity given.
func NewQueryCache(capacity int) *QueryCache {
	if capacity <= 0 {
		capacity = defaultCacheCapacity
	}
	return &QueryCache{
		capacity: capacity,
		entries:  make(map[cacheKey]*cacheEntry, capacity),
	}
}

// Get returns the cached vector for the recipe and query.
func (c *QueryCache) Get(recipe, query string) ([]float32, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	k := cacheKey{recipe: recipe, query: query}
	entry, ok := c.entries[k]
	if !ok {
		return nil, false
	}
	c.moveToFront(entry)
	return entry.vector, true
}

// Put stores the vector for the recipe and query.
func (c *QueryCache) Put(recipe, query string, vector []float32) {
	if c == nil || len(vector) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	k := cacheKey{recipe: recipe, query: query}
	if entry, ok := c.entries[k]; ok {
		c.moveToFront(entry)
		entry.vector = vector
		return
	}

	if len(c.entries) >= c.capacity && c.tail != nil {
		old := c.tail
		c.remove(old)
		delete(c.entries, old.key)
	}

	entry := &cacheEntry{key: k, vector: vector}
	c.pushFront(entry)
	c.entries[k] = entry
}

// Count returns how many entries the cache holds.
func (c *QueryCache) Count() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func (c *QueryCache) moveToFront(e *cacheEntry) {
	if c.head == e {
		return
	}
	c.remove(e)
	c.pushFront(e)
}

func (c *QueryCache) pushFront(e *cacheEntry) {
	e.prev = nil
	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
	if c.tail == nil {
		c.tail = e
	}
}

func (c *QueryCache) remove(e *cacheEntry) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		c.head = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		c.tail = e.prev
	}
	e.prev = nil
	e.next = nil
}
