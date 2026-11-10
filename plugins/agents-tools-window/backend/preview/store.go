// Package preview keeps the edit previews the assistant proposed and not yet answered.
package preview

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Clock gets the current time.
type Clock interface {
	Now() time.Time
}

// Preview is a proposed edit of a note.
type Preview struct {
	ID           string
	Path         string
	ProposedText string
	OriginalText string
	Diff         string
	Additions    int
	Deletions    int
	CreatedAt    time.Time
}

// Store keeps previews by identifier.
type Store struct {
	mu       sync.RWMutex
	previews map[string]Preview
	seq      atomic.Uint64
	clock    Clock
}

// NewStore creates an empty store whose identifiers and creation times come from clock.
func NewStore(clock Clock) *Store {
	return &Store{
		mu:       sync.RWMutex{},
		previews: make(map[string]Preview),
		seq:      atomic.Uint64{},
		clock:    clock,
	}
}

// Save stores a preview under its identifier.
func (s *Store) Save(preview Preview) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.previews[preview.ID] = preview
}

// Get returns the preview with the identifier.
func (s *Store) Get(id string) (Preview, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.previews[id]
	return p, ok
}

// List returns all stored previews.
func (s *Store) List() []Preview {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Preview, 0, len(s.previews))
	for _, p := range s.previews {
		result = append(result, p)
	}
	return result
}

// Clear removes all stored previews.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.previews = make(map[string]Preview)
}

// NextID returns a new unique identifier.
func (s *Store) NextID() string {
	seq := s.seq.Add(1)
	return fmt.Sprintf("prev-%d-%d", s.clock.Now().UnixNano(), seq)
}

// Now returns the time of the store's clock.
func (s *Store) Now() time.Time {
	return s.clock.Now()
}
