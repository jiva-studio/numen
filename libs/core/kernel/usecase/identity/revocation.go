package identity

import (
	"context"
	"fmt"
	"io"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const maxRevokeAttempts = 3

// revocation makes the chain that revokes a device and goes on with it when the server answers.
type revocation struct {
	records records
	clock   Clock
	random  io.Reader
}

func (r revocation) makeChain(state identitylib.State, device []byte) (identitylib.Chain, error) {
	keys, err := state.OpenAccountKeys()
	if err != nil {
		return identitylib.Chain{}, err
	}
	chain, _, err := state.Chain.RevokeDevice(state.Record.Device, keys, device, r.random, r.clock.Now())
	return chain, err
}

func (r revocation) revokeWith(state identitylib.State, keys identitylib.AccountKeys, device []byte) (identitylib.Chain, bool) {
	chain, _, err := state.Chain.RevokeDevice(state.Record.Device, keys, device, r.random, r.clock.Now())
	return chain, err == nil
}

func (r revocation) answerLocked(ctx context.Context, store *linkStore, accepted bool, latest []byte) error {
	pending := store.revoke
	state, err := r.records.readState(ctx)
	if err != nil {
		return err
	}
	if accepted {
		store.revoke = nil
		_, err := r.records.adoptState(ctx, state, pending.chain)
		return err
	}
	head, err := identitylib.ParseChain(latest)
	if err != nil {
		return fmt.Errorf("the chain of the server does not parse: %w", err)
	}
	isRaced := head.IsSignedSince(state.Chain, pending.device)
	next, err := r.records.adoptState(ctx, state, head)
	if err != nil {
		store.revoke = nil
		return err
	}
	if isRaced || pending.attempts >= maxRevokeAttempts {
		store.revoke = nil
		return identitydomain.ErrRevokeRaced
	}
	return r.rebuildLocked(store, next, pending)
}

func (r revocation) rebuildLocked(store *linkStore, state identitylib.State, pending *pendingRevoke) error {
	keys, err := state.OpenAccountKeys()
	if err != nil {
		return err
	}
	chain, isRevoked := r.revokeWith(state, keys, pending.device)
	if !isRevoked {
		store.revoke = nil
		return nil
	}
	pending.chain = chain
	pending.attempts++
	store.bumpLocked()
	return nil
}
