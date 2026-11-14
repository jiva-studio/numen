// Package searchcancel holds the cancels of the running searches of a window, by the id the window gave
// each search.
package searchcancel

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// The window sends a cancel on a request of its own, which can reach the core before the search it
// cancels. Such a cancel is kept for earlyCancelLife, and at most earlyCancelLimit of them.
const (
	earlyCancelLife  = 5 * time.Second
	earlyCancelLimit = 256
)

// Searches holds the cancel of each running search that has an id, and the ids canceled before their
// search started. A search that starts with such an id starts canceled. A search without an id cannot be
// canceled. It is safe to use from several goroutines.
type Searches struct {
	clock machineport.Clock

	mu      sync.Mutex
	running map[string]context.CancelFunc
	early   map[string]time.Time
}

// NewSearches returns the searches with no search running. clock times the cancels that arrive before
// their search.
func NewSearches(clock machineport.Clock) *Searches {
	return &Searches{clock: clock, running: map[string]context.CancelFunc{}, early: map[string]time.Time{}}
}

// Start returns the context the search with the id runs in, and the function that ends the search.
func (s *Searches) Start(ctx context.Context, id string) (context.Context, func()) {
	if id == "" {
		return ctx, func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.dropExpired()
	if _, isCanceled := s.early[id]; isCanceled {
		delete(s.early, id)
		s.mu.Unlock()
		cancel()
		return ctx, func() {}
	}
	s.running[id] = cancel
	s.mu.Unlock()
	return ctx, func() {
		s.mu.Lock()
		delete(s.running, id)
		s.mu.Unlock()
		cancel()
	}
}

// Cancel cancels the running search with the id, or else the search with the id that starts within
// earlyCancelLife.
func (s *Searches) Cancel(id string) {
	if id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancel, isRunning := s.running[id]; isRunning {
		cancel()
		return
	}
	s.dropExpired()
	if len(s.early) >= earlyCancelLimit {
		s.dropOldest()
	}
	s.early[id] = s.clock.Now()
}

func (s *Searches) dropExpired() {
	now := s.clock.Now()
	for id, at := range s.early {
		if now.Sub(at) >= earlyCancelLife {
			delete(s.early, id)
		}
	}
}

func (s *Searches) dropOldest() {
	oldestID, oldestAt := "", time.Time{}
	for id, at := range s.early {
		if oldestID == "" || at.Before(oldestAt) {
			oldestID, oldestAt = id, at
		}
	}
	delete(s.early, oldestID)
}
