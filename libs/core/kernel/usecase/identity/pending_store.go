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

// pendingStore keeps the one restore that waits for the account chain of a sync plugin. Its keys stay in memory.
type pendingStore struct {
	mu      sync.Mutex
	current *pendingRestore
}

func newPendingStore() *pendingStore { return &pendingStore{} }

func (s *pendingStore) replace(next *pendingRestore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearLocked()
	s.current = next
}

func (s *pendingStore) clearLocked() {
	if s.current != nil {
		close(s.current.replaced)
		s.current = nil
	}
}

func (s *pendingStore) isCurrent(restore *pendingRestore) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current == restore
}

func (s *pendingStore) read() *pendingRestore {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}
