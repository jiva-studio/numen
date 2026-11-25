// Package streams holds the open streams of change hints.
package streams

import (
	"maps"
	"sync"
	"time"
)

// Watcher is one open stream: the latest seq of each vault that changed and was not sent yet, a Signal of
// capacity one that wakes the stream, and an End channel that ends it.
type Watcher struct {
	account string
	pending map[string]uint64
	Signal  chan struct{}
	End     chan struct{}
}

// Set holds the open streams. Its lock is taken after the lock of the server, never before.
type Set struct {
	mu        sync.Mutex
	Heartbeat time.Duration
	watchers  map[*Watcher]struct{}
}

// New returns a set of streams whose silence ends in a heartbeat after the duration.
func New(heartbeat time.Duration) *Set {
	return &Set{Heartbeat: heartbeat, watchers: map[*Watcher]struct{}{}}
}

// Add opens a stream for the account.
func (s *Set) Add(account string) *Watcher {
	s.mu.Lock()
	defer s.mu.Unlock()
	w := &Watcher{account: account, pending: map[string]uint64{}, Signal: make(chan struct{}, 1), End: make(chan struct{})}
	s.watchers[w] = struct{}{}
	return w
}

// Remove closes the stream of the watcher.
func (s *Set) Remove(w *Watcher) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.watchers, w)
}

// Count returns how many streams are open.
func (s *Set) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.watchers)
}

// EndAll ends every open stream.
func (s *Set) EndAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for w := range s.watchers {
		close(w.End)
	}
	clear(s.watchers)
}

// Merge adds the seqs to what the watcher has not sent yet.
func (s *Set) Merge(w *Watcher, seqs map[string]uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for vault, seq := range seqs {
		w.pending[vault] = max(w.pending[vault], seq)
	}
	wake(w)
}

// Announce adds the seq of the vault to every stream of the account.
func (s *Set) Announce(account, vault string, seq uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for w := range s.watchers {
		if w.account != account {
			continue
		}
		w.pending[vault] = max(w.pending[vault], seq)
		wake(w)
	}
}

// Take returns what the watcher has not sent yet and clears it.
func (s *Set) Take(w *Watcher) map[string]uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	taken := maps.Clone(w.pending)
	clear(w.pending)
	return taken
}

func wake(w *Watcher) {
	select {
	case w.Signal <- struct{}{}:
	default:
	}
}
