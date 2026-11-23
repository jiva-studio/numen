package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// DecideLink is the answer of the person on the host to the new device.
type DecideLink struct {
	links linkHold
	host  linkHost
}

// NewDecideLink returns the use case that answers the new device.
func NewDecideLink(d Deps) *DecideLink {
	return &DecideLink{links: d.progress.links, host: d.getHost()}
}

// Execute declines the new device, or approves it. After an approval in mode show the chain that lists the device goes to the server; in mode
// scan the new device is asked for the code first. It returns ErrNoLink when no new device waits for an answer.
func (u *DecideLink) Execute(ctx context.Context, approve bool) error {
	store, release := u.links.hold()
	defer release()
	store.expireLocked(u.host.clock.Now())
	link := store.current
	if link == nil || link.progress.Role != identitydomain.LinkRoleHost || link.progress.Phase != identitydomain.LinkPhaseConfirmPeer || link.progress.IsApproved {
		return identitydomain.ErrNoLink
	}
	if !approve {
		store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonDeclined, 0))
		store.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonDeclined)
		return nil
	}
	if link.progress.Mode == identitydomain.LinkModeShow {
		return u.host.publishLocked(ctx, store, link)
	}
	link.progress.IsApproved = true
	store.sendLocked(link, encodeAccept(true))
	return nil
}
