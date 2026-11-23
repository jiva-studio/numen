package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// PutLinkInbound hands the core a blob the other side posted to the relay.
type PutLinkInbound struct{ deps Deps }

// NewPutLinkInbound returns the use case that takes a blob from the courier.
func NewPutLinkInbound(d Deps) *PutLinkInbound { return &PutLinkInbound{deps: d} }

// Execute takes a blob that came from the relay. A blob that does not authenticate is dropped, so someone who knows the channel
// cannot end the link; ErrNoLink means no link was started. A link that ended takes no blob.
func (u *PutLinkInbound) Execute(ctx context.Context, blob []byte) error {
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expireLocked(u.deps.clock.Now())
	link := store.current
	if link == nil {
		return identitydomain.ErrNoLink
	}
	if link.isEnded() {
		return nil
	}
	if link.hs.channel == nil {
		u.deps.completeHandshakeLocked(link, blob)
		return nil
	}
	plain, err := link.hs.channel.Open(blob)
	if err != nil || len(plain) == 0 {
		return nil
	}
	if link.role == identitydomain.LinkRoleHost {
		return u.deps.receiveAsHostLocked(ctx, link, plain[0], plain[1:])
	}
	return u.deps.receiveAsJoinerLocked(link, plain[0], plain[1:])
}

func (d Deps) completeHandshakeLocked(link *pendingLink, blob []byte) {
	store := d.links
	switch {
	case link.hs.displayer != nil:
		reply, channel, err := link.hs.displayer.Accept(blob)
		if err != nil {
			return
		}
		link.hs.channel = channel
		store.queueLocked(link, reply)
	case link.hs.scanner != nil:
		channel, err := link.hs.scanner.Accept(blob)
		if err != nil {
			return
		}
		link.hs.channel = channel
	default:
		return
	}
	if link.role == identitydomain.LinkRoleJoiner {
		store.sendLocked(link, encodeJoin(link.joiner.device.Public(), link.own.name))
		store.setPhaseLocked(link, identitydomain.LinkPhaseWaitingForAccept)
	}
}
