package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// PutPublishedChain takes the answer of the server to the chain the courier published.
type PutPublishedChain struct {
	links      linkHold
	host       linkHost
	revocation revocation
}

// NewPutChainPublished returns the use case that takes the answer of the server.
func NewPutChainPublished(d Deps) *PutPublishedChain {
	return &PutPublishedChain{links: d.progress.links, host: d.getHost(), revocation: d.getRevocation()}
}

// Execute goes on with the link or the revoke that waited: when the server accepted the chain it is kept, the host sends the grant and a
// revoke takes its new keys. When another device appended first, latest is the chain the server holds: it is adopted and the link is built
// once more on it. A revoke is built again as long as the other devices only added devices, and ends with ErrRevokeRaced when the device
// to revoke signed an entry meanwhile. It returns ErrNoLink when nothing waits for the server.
func (u *PutPublishedChain) Execute(ctx context.Context, accepted bool, latest []byte) error {
	store, release := u.links.hold()
	defer release()
	store.expireLocked(u.host.clock.Now())
	switch link := store.current; {
	case link != nil && link.progress.Phase == identitydomain.LinkPhasePublishing:
		return u.host.answerLocked(ctx, store, link, accepted, latest)
	case store.revoke != nil:
		return u.revocation.answerLocked(ctx, store, accepted, latest)
	}
	return identitydomain.ErrNoLink
}
