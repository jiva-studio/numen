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

// Execute declines the new device, or approves it. After an approval the new device is asked for the code the host shows,
// in either mode, and only the right code sends the chain that lists the device to the server. It returns ErrNoLink when no new device waits for an answer.
func (u *DecideLink) Execute(_ context.Context, approve bool) error {
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
	link.progress.IsApproved = true
	store.sendLocked(link, encodeAccept(true))
	return nil
}
