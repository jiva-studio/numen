package identity

import (
	"sync"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const (
	maxLinkWait = 20 * time.Second
)

// pendingLink is the link in progress. peer and chain are of the host: the device that joins and the chain that
// lists it. record and chain are of the new device: its device key with, once granted, the account keys wrapped for
// it, and the chain of the account.
type pendingLink struct {
	progress identitydomain.LinkProgress
	outbox   identitydomain.LinkOutbox
	end      linkEnd
	peer     identitylib.DevicePublicKey
	chain    identitylib.Chain
	record   identitylib.Record
}

type pendingRevoke struct {
	chain    identitylib.Chain
	device   []byte
	attempts int
}

// linkHold is the link in progress and the revoke that waits for the server, which the use cases take under the lock
// of the store: hold waits for the lock and returns the store with the function that gives it back.
type linkHold interface {
	hold() (*linkStore, func())
}

// linkStore keeps the one link in progress and the revoke that waits for the server. Its keys stay in memory. Each
// change takes the next version, which the courier waits on.
type linkStore struct {
	mu      sync.Mutex
	ticker  versionClock
	current *pendingLink
	revoke  *pendingRevoke
	signal  changeNotifier
}

func newLinkStore(signal *changeSignal) *linkStore {
	return &linkStore{ticker: &versionTicker{changed: make(chan struct{})}, signal: signal}
}

func (s *linkStore) hold() (*linkStore, func()) {
	s.mu.Lock()
	return s, s.mu.Unlock
}

func (s *linkStore) bumpLocked() uint64 {
	version := s.ticker.bump()
	s.signal.notify()
	return version
}

func (s *linkStore) expireLocked(now time.Time) {
	if l := s.current; l != nil && l.progress.IsExpired(now) {
		s.endLocked(l, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonExpired)
	}
}

func (s *linkStore) queueLocked(l *pendingLink, blob []byte) {
	l.outbox.Items = append(l.outbox.Items, identitydomain.LinkBlob{Version: s.bumpLocked(), Blob: blob})
}

func (s *linkStore) sendLocked(l *pendingLink, plain []byte) {
	if l.end != nil && l.end.isOpen() {
		s.queueLocked(l, l.end.seal(plain))
	}
}

func (s *linkStore) endLocked(l *pendingLink, phase identitydomain.LinkPhase, reason identitydomain.LinkReason) {
	l.progress.Phase, l.progress.Reason, l.outbox.IsClosed = phase, reason, true
	s.bumpLocked()
}

func (s *linkStore) setPhaseLocked(l *pendingLink, phase identitydomain.LinkPhase) {
	l.progress.Phase = phase
	s.bumpLocked()
}

func (s *linkStore) getCurrentLocked() (*pendingLink, error) {
	if s.current == nil || s.current.progress.Phase == identitydomain.LinkPhaseCanceled {
		return nil, identitydomain.ErrNoLink
	}
	return s.current, nil
}

func (l *pendingLink) isEnded() bool { return l.progress.IsEnded() }

// versionClock numbers the changes of a store and lets a waiter learn of the next one.
type versionClock interface {
	bump() uint64
	getVersion() uint64
	getChangeSignal() <-chan struct{}
}

type versionTicker struct {
	version uint64
	changed chan struct{}
}

func (t *versionTicker) bump() uint64 {
	t.version++
	close(t.changed)
	t.changed = make(chan struct{})
	return t.version
}

func (t *versionTicker) getVersion() uint64 { return t.version }

func (t *versionTicker) getChangeSignal() <-chan struct{} { return t.changed }
