package identity

import (
	"sync"
	"time"

	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

type pendingRestore struct {
	recovery  identitylib.RecoveryKey
	device    identitylib.DeviceKey
	startedAt time.Time
	replaced  chan struct{}
}

// PendingStore keeps the one restore that waits for the account chain of a sync plugin. Its keys stay in memory.
type PendingStore struct {
	mu sync.Mutex
	// write serializes the check for an absent identity and the write of one across every window, which share this store.
	write   sync.Mutex
	current *pendingRestore
}

// NewPendingStore returns a store with no restore pending.
func NewPendingStore() *PendingStore { return &PendingStore{} }

func (s *PendingStore) replace(next *pendingRestore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearLocked()
	s.current = next
}

func (s *PendingStore) clearLocked() {
	if s.current != nil {
		close(s.current.replaced)
		s.current = nil
	}
}

func (s *PendingStore) isCurrent(restore *pendingRestore) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current == restore
}

func (s *PendingStore) read() *pendingRestore {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}
