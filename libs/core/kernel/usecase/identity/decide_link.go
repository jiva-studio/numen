package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// DecideLink is the answer of the person on the host to the new device.
type DecideLink struct{ deps Deps }

// NewDecideLink returns the use case that answers the new device.
func NewDecideLink(d Deps) *DecideLink { return &DecideLink{deps: d} }

// Execute declines the new device, or approves it. After an approval in mode show the chain that lists the device goes to the server; in mode
// scan the new device is asked for the code first. It returns ErrNoLink when no new device waits for an answer.
func (u *DecideLink) Execute(ctx context.Context, approve bool) error {
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expireLocked(u.deps.clock.Now())
	link := store.current
	if link == nil || link.role != identitydomain.LinkRoleHost || link.phase != identitydomain.LinkPhaseConfirmPeer || link.host.isApproved {
		return identitydomain.ErrNoLink
	}
	if !approve {
		store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonDeclined, 0))
		store.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonDeclined)
		return nil
	}
	if link.mode == identitydomain.LinkModeShow {
		return u.deps.publishLinkLocked(ctx, link)
	}
	link.host.isApproved = true
	store.sendLocked(link, encodeAccept(true))
	return nil
}
