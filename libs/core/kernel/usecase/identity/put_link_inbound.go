package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// PutLinkInbound hands the core a blob the other side posted to the relay.
type PutLinkInbound struct {
	links linkHold
	clock machineport.InstallationClock
	host  linkHost
}

// NewPutLinkInbound returns the use case that takes a blob from the courier.
func NewPutLinkInbound(d Deps) *PutLinkInbound {
	return &PutLinkInbound{links: d.progress.links, clock: d.clock, host: d.getHost()}
}

// Execute takes a blob that came from the relay. A blob that does not authenticate is dropped, so someone who knows the channel
// cannot end the link; ErrNoLink means no link was started. A link that ended takes no blob.
func (u *PutLinkInbound) Execute(ctx context.Context, blob []byte) error {
	store, release := u.links.hold()
	defer release()
	store.expireLocked(u.clock.Now())
	link := store.current
	if link == nil {
		return identitydomain.ErrNoLink
	}
	if link.isEnded() {
		return nil
	}
	if link.end == nil || !link.end.isOpen() {
		completeHandshakeLocked(store, link, blob)
		return nil
	}
	plain, err := link.end.open(blob)
	if err != nil || len(plain) == 0 {
		return nil
	}
	if link.progress.Role == identitydomain.LinkRoleHost {
		return u.host.receiveLocked(store, link, plain[0], plain[1:])
	}
	receiveAsJoinerLocked(store, link, plain[0], plain[1:])
	return nil
}

func completeHandshakeLocked(store *linkStore, link *pendingLink, blob []byte) {
	if link.end == nil {
		return
	}
	reply, err := link.end.accept(blob)
	if err != nil {
		return
	}
	if reply != nil {
		store.queueLocked(link, reply)
	}
	if link.progress.Role == identitydomain.LinkRoleJoiner {
		store.sendLocked(link, encodeJoin(link.record.Device.Public(), link.progress.DeviceName))
		store.setPhaseLocked(link, identitydomain.LinkPhaseWaitingForAccept)
	}
}
