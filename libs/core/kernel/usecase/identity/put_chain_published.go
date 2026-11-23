package identity

import (
	"context"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const maxRevokeAttempts = 3

// PutChainPublished takes the answer of the server to the chain the courier published.
type PutChainPublished struct{ deps Deps }

// NewPutChainPublished returns the use case that takes the answer of the server.
func NewPutChainPublished(d Deps) *PutChainPublished { return &PutChainPublished{deps: d} }

// Execute goes on with the link or the revoke that waited: when the server accepted the chain it is kept, the host sends the grant and a
// revoke takes its new keys. When another device appended first, latest is the chain the server holds: it is adopted and the link is built
// once more on it. A revoke is built again as long as the other devices only added devices, and ends with ErrRevokeRaced when the device
// to revoke signed an entry meanwhile. It returns ErrNoLink when nothing waits for the server.
func (u *PutChainPublished) Execute(ctx context.Context, accepted bool, latest []byte) error {
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expireLocked(u.deps.clock.Now())
	switch link := store.current; {
	case link != nil && link.phase == identitydomain.LinkPhasePublishing:
		return u.deps.answerLinkLocked(ctx, link, accepted, latest)
	case store.revoke != nil:
		return u.deps.answerRevokeLocked(ctx, accepted, latest)
	}
	return identitydomain.ErrNoLink
}

func (d Deps) answerLinkLocked(ctx context.Context, link *pendingLink, accepted bool, latest []byte) error {
	state, err := d.readState(ctx)
	if err != nil {
		return err
	}
	if accepted {
		return d.grantLocked(ctx, link, state)
	}
	state, isAdopted := d.adoptLatest(ctx, state, latest)
	if !isAdopted || link.host.isRebuilt || state.Status != identitylib.StatusActive {
		d.links.sendLocked(link, encodeRefuse(identitydomain.LinkReasonStaleChain, 0))
		d.links.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonStaleChain)
		return nil
	}
	link.host.isRebuilt = true
	return d.extendLocked(link, state)
}

func (d Deps) grantLocked(ctx context.Context, link *pendingLink, state identitylib.State) error {
	next, err := d.adoptState(ctx, state, link.host.extended)
	if err != nil {
		return err
	}
	keys, err := next.OpenAccountKeys()
	if err != nil {
		return err
	}
	wrapped, err := keys.WrapForDevice(link.host.join.device)
	if err != nil {
		return err
	}
	d.links.sendLocked(link, encodeGrant(wrapped.Bytes(), next.Chain.Bytes()))
	d.links.setPhaseLocked(link, identitydomain.LinkPhaseConfirmAccount)
	return nil
}

func (d Deps) answerRevokeLocked(ctx context.Context, accepted bool, latest []byte) error {
	store := d.links
	pending := store.revoke
	state, err := d.readState(ctx)
	if err != nil {
		return err
	}
	if accepted {
		store.revoke = nil
		_, err := d.adoptState(ctx, state, pending.chain)
		return err
	}
	head, err := identitylib.ParseChain(latest)
	if err != nil {
		return fmt.Errorf("the chain of the server does not parse: %w", err)
	}
	isRaced := head.IsSignedSince(state.Chain, pending.device)
	next, err := d.adoptState(ctx, state, head)
	if err != nil {
		store.revoke = nil
		return err
	}
	if isRaced || pending.attempts >= maxRevokeAttempts {
		store.revoke = nil
		return identitydomain.ErrRevokeRaced
	}
	return d.rebuildRevokeLocked(next, pending)
}

func (d Deps) rebuildRevokeLocked(state identitylib.State, pending *pendingRevoke) error {
	keys, err := state.OpenAccountKeys()
	if err != nil {
		return err
	}
	chain, isRevoked := d.revokeChain(state, keys, pending)
	if !isRevoked {
		d.links.revoke = nil
		return nil
	}
	pending.chain = chain
	pending.attempts++
	d.links.bumpLocked()
	return nil
}

func (d Deps) adoptLatest(ctx context.Context, state identitylib.State, latest []byte) (identitylib.State, bool) {
	head, err := identitylib.ParseChain(latest)
	if err != nil {
		return state, false
	}
	adopted, err := d.adoptState(ctx, state, head)
	return adopted, err == nil
}

func (d Deps) revokeChain(state identitylib.State, keys identitylib.AccountKeys, pending *pendingRevoke) (identitylib.Chain, bool) {
	chain, _, err := state.Chain.RevokeDevice(state.Record.Device, keys, pending.device, d.random, d.clock.Now())
	return chain, err == nil
}
