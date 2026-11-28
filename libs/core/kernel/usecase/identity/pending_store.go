package identity

import (
	"sync"
	"time"

	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

type pendingRestore struct {
	recovery  identitylib.RecoveryKey
	device    identitylib.DeviceKey
	mark      identitylib.Mark
	startedAt time.Time
	replaced  chan struct{}
	refusal   error
}

// restoreHold changes the restore that waits for the account chain of a sync plugin: replace, isCurrent and expire take the
// lock of the store themselves, and hold waits for the lock and returns the store with the function that gives it back.
type restoreHold interface {
	replace(next *pendingRestore)
	isCurrent(restore *pendingRestore) bool
	expire(restore *pendingRestore)
	hold() (*pendingStore, func())
}

// restoreReader reads the restore that waits for the account chain of a sync plugin; read and readRefusal take the lock of the
// store themselves.
type restoreReader interface {
	read() *pendingRestore
	readRefusal() error
}

// pendingStore keeps the one restore that waits for the account chain of a sync plugin. Its keys stay in memory.
type pendingStore struct {
	mu      sync.Mutex
	current *pendingRestore
	signal  *changeSignal
}

func newPendingStore(signal *changeSignal) *pendingStore { return &pendingStore{signal: signal} }

func (s *pendingStore) hold() (*pendingStore, func()) {
	s.mu.Lock()
	return s, s.mu.Unlock
}

func (s *pendingStore) replace(next *pendingRestore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearLocked()
	s.current = next
	s.signal.notify()
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
	return s.current == restore && s.current.refusal == nil
}

func (s *pendingStore) read() *pendingRestore {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil || s.current.refusal != nil {
		return nil
	}
	return s.current
}

func (s *pendingStore) readRefusal() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return nil
	}
	return s.current.refusal
}

func (s *pendingStore) refuseLocked(restore *pendingRestore, reason error) {
	restore.recovery = identitylib.RecoveryKey{}
	restore.device = identitylib.DeviceKey{}
	restore.refusal = reason
	s.signal.notify()
}

func (s *pendingStore) expire(restore *pendingRestore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == restore {
		s.clearLocked()
		restore.recovery = identitylib.RecoveryKey{}
		restore.device = identitylib.DeviceKey{}
		s.signal.notify()
	}
}
