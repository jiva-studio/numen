package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// GetChainPublish tells the courier which chain the server must hold.
type GetChainPublish struct {
	links linkHold
	clock machineport.InstallationClock
}

// NewGetChainPublish returns the use case that tells the courier which chain to publish.
func NewGetChainPublish(d Deps) *GetChainPublish {
	return &GetChainPublish{links: d.progress.links, clock: d.clock}
}

// Execute returns the chain a link or a revoke waits to have accepted, or none.
func (u *GetChainPublish) Execute(_ context.Context) (identitydomain.ChainPublish, error) {
	store, release := u.links.hold()
	defer release()
	store.expireLocked(u.clock.Now())
	switch link := store.current; {
	case link != nil && link.progress.Phase == identitydomain.LinkPhasePublishing:
		return identitydomain.ChainPublish{IsPending: true, Chain: link.chain.Bytes()}, nil
	case store.revoke != nil:
		return identitydomain.ChainPublish{IsPending: true, Chain: store.revoke.chain.Bytes()}, nil
	}
	return identitydomain.ChainPublish{}, nil
}
