// Package searchstore keeps the searches of the core in memory, each with a channel closed when it stops
// running.
package searchstore

import (
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
)

type entry struct {
	record  searchrun.Record
	stopped chan struct{}
}

// Store is the searches by id. It is safe to use from several goroutines.
type Store struct {
	mu      sync.Mutex
	entries map[searchrun.ID]*entry
}

// New returns an empty store.
func New() *Store {
	return &Store{entries: map[searchrun.ID]*entry{}}
}

func (s *Store) AddSearch(r searchrun.Record) <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := &entry{record: r, stopped: make(chan struct{})}
	if !r.IsRunning {
		close(e.stopped)
	}
	s.entries[r.ID] = e
	return e.stopped
}

func (s *Store) FindSearch(id searchrun.ID) (searchrun.Record, <-chan struct{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, isFound := s.entries[id]
	if !isFound {
		return searchrun.Record{}, nil, false
	}
	return e.record, e.stopped, true
}

func (s *Store) ChangeSearch(id searchrun.ID, change func(*searchrun.Record)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, isFound := s.entries[id]
	if !isFound {
		return false
	}
	wasRunning := e.record.IsRunning
	change(&e.record)
	if wasRunning && !e.record.IsRunning {
		close(e.stopped)
	}
	return true
}

func (s *Store) RemoveSearches(ids []searchrun.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		delete(s.entries, id)
	}
}

func (s *Store) ListSearches() []searchrun.Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	records := make([]searchrun.Record, 0, len(s.entries))
	for _, e := range s.entries {
		records = append(records, e.record)
	}
	return records
}
