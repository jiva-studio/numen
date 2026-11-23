package identity

import (
	"slices"
	"sync"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const (
	linkLifetime = 10 * time.Minute
	maxLinkWait  = 20 * time.Second
	maxCodeTries = 3
	codeDigits   = 6
)

type outboxItem struct {
	version uint64
	blob    []byte
}

// handshake is the key exchange of one link: the side that shows the QR code holds a displayer, the side that reads it a scanner.
type handshake struct {
	displayer *identitylib.LinkDisplayer
	scanner   *identitylib.LinkScanner
	channel   *identitylib.LinkChannel
}

type joinRequest struct {
	device identitylib.DevicePublicKey
	name   string
}

// hostSide is what the device that has the account holds about the new device.
type hostSide struct {
	join       joinRequest
	code       string
	triesLeft  int
	isApproved bool
	isRebuilt  bool
	extended   identitylib.Chain
}

// joinerSide is what the new device holds until the person confirms the account.
type joinerSide struct {
	device    identitylib.DeviceKey
	wrapped   identitylib.WrappedAccountKeys
	chain     identitylib.Chain
	triesLeft int
}

type ownIdentity struct {
	name        string
	fingerprint string
	account     string
}

// relayState is what a link keeps of its channel on the relay and of the blobs the courier has not yet carried.
type relayState struct {
	server       string
	channelID    []byte
	wantsChannel bool
	isClosed     bool
	items        []outboxItem
}

type pendingLink struct {
	role      identitydomain.LinkRole
	mode      identitydomain.LinkMode
	phase     identitydomain.LinkPhase
	reason    identitydomain.LinkReason
	startedAt time.Time
	qrText    string
	relay     relayState
	own       ownIdentity
	hs        handshake
	host      hostSide
	joiner    joinerSide
}

// LinkStore keeps the one link in progress and the revoke that waits for the server. Its keys stay in memory.
type LinkStore struct {
	mu      sync.Mutex
	version uint64
	changed chan struct{}
	current *pendingLink
	revoke  *pendingRevoke
}

// NewLinkStore returns a store with no link and no revoke pending.
func NewLinkStore() *LinkStore { return &LinkStore{changed: make(chan struct{})} }

// bumpLocked records a change: a new version and a wake-up for every waiting courier.
func (s *LinkStore) bumpLocked() uint64 {
	s.version++
	close(s.changed)
	s.changed = make(chan struct{})
	return s.version
}

func (l *pendingLink) isEnded() bool {
	return slices.Contains([]identitydomain.LinkPhase{
		identitydomain.LinkPhaseFailed, identitydomain.LinkPhaseLinked, identitydomain.LinkPhaseCancelled,
	}, l.phase)
}

// expireLocked ends the link that ran for its whole lifetime.
func (s *LinkStore) expireLocked(now time.Time) {
	if l := s.current; l != nil && !l.isEnded() && !now.Before(l.startedAt.Add(linkLifetime)) {
		s.endLocked(l, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonExpired)
	}
}

func (s *LinkStore) queueLocked(l *pendingLink, blob []byte) {
	l.relay.items = append(l.relay.items, outboxItem{version: s.bumpLocked(), blob: blob})
}

// sendLocked seals a message of the application for the other side; before the handshake there is nothing to seal it with.
func (s *LinkStore) sendLocked(l *pendingLink, plain []byte) {
	if l.hs.channel != nil {
		s.queueLocked(l, l.hs.channel.Seal(plain))
	}
}

func (s *LinkStore) endLocked(l *pendingLink, phase identitydomain.LinkPhase, reason identitydomain.LinkReason) {
	l.phase, l.reason, l.relay.isClosed = phase, reason, true
	s.bumpLocked()
}

func (s *LinkStore) setPhaseLocked(l *pendingLink, phase identitydomain.LinkPhase) {
	l.phase = phase
	s.bumpLocked()
}

// acknowledge drops the blobs the courier has carried: those of a version up to the one it names.
func (l *pendingLink) acknowledge(version uint64) {
	l.relay.items = slices.DeleteFunc(l.relay.items, func(item outboxItem) bool { return item.version <= version })
}

type pendingRevoke struct {
	chain    identitylib.Chain
	device   []byte
	attempts int
}
