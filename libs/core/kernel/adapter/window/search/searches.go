package search

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

// searches holds the cancel of each running search that carries an id, and the ids cancelled before
// their search started. A search that starts with such an id starts cancelled. A search without an id
// cannot be cancelled.
type searches struct {
	clock machineport.Clock

	mu      sync.Mutex
	running map[string]context.CancelFunc
	early   map[string]time.Time
}

func newSearches(clock machineport.Clock) *searches {
	return &searches{clock: clock, running: map[string]context.CancelFunc{}, early: map[string]time.Time{}}
}

func (s *searches) start(ctx context.Context, id string) (context.Context, func()) {
	if id == "" {
		return ctx, func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.dropExpired()
	if _, isCancelled := s.early[id]; isCancelled {
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

func (s *searches) cancel(id string) {
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

func (s *searches) dropExpired() {
	now := s.clock.Now()
	for id, at := range s.early {
		if now.Sub(at) >= earlyCancelLife {
			delete(s.early, id)
		}
	}
}

func (s *searches) dropOldest() {
	oldestID, oldestAt := "", time.Time{}
	for id, at := range s.early {
		if oldestID == "" || at.Before(oldestAt) {
			oldestID, oldestAt = id, at
		}
	}
	delete(s.early, oldestID)
}
