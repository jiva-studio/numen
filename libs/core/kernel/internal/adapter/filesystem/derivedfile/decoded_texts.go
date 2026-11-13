package derivedfile

import (
	"container/list"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// DefaultDecodedBudget is how many bytes of decoded text DecodedTexts keeps by default.
const DefaultDecodedBudget = 64 << 20

// segmentOverhead is what one segment takes besides its strings: the three string headers.
const segmentOverhead = 48

// DecodedTexts keeps the segments of the text files decoded last, by the hash of the file's content, up
// to a budget of bytes. Equal content decodes to equal segments, so an entry never goes stale; a file
// whose content changed has another hash. The segments it returns are shared and must not be changed.
// A nil DecodedTexts keeps nothing.
type DecodedTexts struct {
	mu      sync.Mutex
	budget  int
	size    int
	order   *list.List
	entries map[string]*list.Element
}

type decodedEntry struct {
	hash     string
	segments []domain.TextSegment
	size     int
}

// NewDecodedTexts returns an empty DecodedTexts that keeps up to budget bytes of segments.
func NewDecodedTexts(budget int) *DecodedTexts {
	return &DecodedTexts{budget: budget, order: list.New(), entries: map[string]*list.Element{}}
}

func (c *DecodedTexts) get(hash string) ([]domain.TextSegment, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	element, ok := c.entries[hash]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(element)
	entry, _ := element.Value.(decodedEntry)
	return slices.Clip(entry.segments), true
}

func (c *DecodedTexts) put(hash string, segments []domain.TextSegment) {
	if c == nil {
		return
	}
	size := getSegmentsSize(segments)
	if size > c.budget {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries[hash]; ok {
		return
	}
	c.entries[hash] = c.order.PushFront(decodedEntry{hash: hash, segments: segments, size: size})
	c.size += size
	for c.size > c.budget {
		oldest := c.order.Back()
		entry, _ := c.order.Remove(oldest).(decodedEntry)
		delete(c.entries, entry.hash)
		c.size -= entry.size
	}
}

func getSegmentsSize(segments []domain.TextSegment) int {
	size := 0
	for _, s := range segments {
		size += segmentOverhead + len(s.Text) + len(s.Heading) + len(s.Selector)
	}
	return size
}
