package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// GetChainPublish tells the courier which chain the server must hold.
type GetChainPublish struct{ deps Deps }

// NewGetChainPublish returns the use case that tells the courier which chain to publish.
func NewGetChainPublish(d Deps) *GetChainPublish { return &GetChainPublish{deps: d} }

// Execute returns the chain a link or a revoke waits to have accepted, or none.
func (u *GetChainPublish) Execute(_ context.Context) (identitydomain.ChainPublish, error) {
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expireLocked(u.deps.clock.Now())
	switch link := store.current; {
	case link != nil && link.phase == identitydomain.LinkPhasePublishing:
		return identitydomain.ChainPublish{IsPending: true, Chain: link.host.extended.Bytes()}, nil
	case store.revoke != nil:
		return identitydomain.ChainPublish{IsPending: true, Chain: store.revoke.chain.Bytes()}, nil
	}
	return identitydomain.ChainPublish{}, nil
}
