package identity

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// CompleteRestore finishes the pending restore with the chain a sync plugin read from its server.
type CompleteRestore struct {
	completion restoreCompletion
	pending    restoreReader
}

// NewCompleteRestore returns the use case that completes a restore.
func NewCompleteRestore(d Deps) *CompleteRestore {
	return &CompleteRestore{completion: d.getCompletion(), pending: d.progress.pending}
}

// Execute joins the account of chainBytes: the chain must verify, have the root of the pending words and contain the chain mark of the
// kit when one was given, the recovery key unwraps the account keys of the latest epoch and they must be the keys of the account box of the
// chain. This device is appended and the recovery key starts a new epoch of keys for every device of the chain, which only the words can
// do, so what the chain wrapped for the words is not the keys in use. Empty chainBytes means the server has no such account, and the
// result is a new identity of the words, unless the kit named a chain mark. The record is written with one SetKey and the pending restore
// is cleared. ErrNoPendingRestore, ErrInvalidChain, ErrIdentityExists and ErrChainOutdated refuse it; after
// ErrChainOutdated the restore is over and GetIdentity reports the refusal.
func (u *CompleteRestore) Execute(ctx context.Context, pluginID string, chainBytes []byte) error {
	if pluginID == "" {
		return errNoPluginID
	}
	restore := u.pending.read()
	if restore == nil {
		return identitydomain.ErrNoPendingRestore
	}
	return u.completion.complete(ctx, restore, chainBytes)
}

// restoreCompletion completes a pending restore: it keeps the identity the restore makes and forgets the restore.
type restoreCompletion struct {
	records  records
	restores restoreHold
	builder  stateBuilder
	gate     CreateGate
}

func (c restoreCompletion) complete(ctx context.Context, restore *pendingRestore, chainBytes []byte) error {
	store, release := c.restores.hold()
	defer release()
	if store.current != restore || restore.refusal != nil {
		return identitydomain.ErrNoPendingRestore
	}
	write, err := c.gate.acquire(ctx)
	if err != nil {
		return fmt.Errorf("wait for the write of the identity: %w", err)
	}
	defer write()
	if err := c.records.checkAbsent(ctx); err != nil {
		return err
	}
	state, err := c.buildRestoredState(ctx, restore, chainBytes)
	if err != nil {
		if errors.Is(err, identitydomain.ErrChainOutdated) {
			store.refuseLocked(restore, err)
		}
		return err
	}
	if err := c.records.writeNewState(ctx, state.Record, state.Chain); err != nil {
		return err
	}
	store.clearLocked()
	return nil
}

func (c restoreCompletion) buildRestoredState(ctx context.Context, restore *pendingRestore, chainBytes []byte) (identitylib.State, error) {
	fingerprint := restore.recovery.Public().Fingerprint().String()
	if len(chainBytes) == 0 {
		if err := c.records.checkHighWater(ctx, fingerprint, 0); err != nil {
			return identitylib.State{}, err
		}
		if !restore.mark.IsZero() {
			return identitylib.State{}, fmt.Errorf("%w: the server holds no chain", identitydomain.ErrChainOutdated)
		}
		return c.builder.buildNew(restore.recovery, restore.device)
	}
	chain, err := identitylib.ParseChain(chainBytes)
	if err != nil {
		return identitylib.State{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidChain, err)
	}
	if !bytes.Equal(chain.Root().Sign, restore.recovery.Public().Sign) {
		return identitylib.State{}, fmt.Errorf("%w: the chain has another root", identitydomain.ErrInvalidChain)
	}
	if err := c.records.checkHighWater(ctx, fingerprint, chain.Len()); err != nil {
		return identitylib.State{}, err
	}
	if !restore.mark.IsZero() && !chain.HasMark(restore.mark) {
		return identitylib.State{}, fmt.Errorf("%w: the chain has %d entries and does not contain the mark %v", identitydomain.ErrChainOutdated, chain.Len(), restore.mark)
	}
	account, err := c.openRecoveryKeys(restore.recovery, chain)
	if err != nil {
		return identitylib.State{}, err
	}
	name := normalizeEntryName(c.builder.names.GetInstanceName())
	now := c.builder.clock.Now()
	chain, err = chain.AppendDevice(restore.recovery, restore.device.Public(), name, now)
	if err != nil {
		return identitylib.State{}, err
	}
	chain, _, err = chain.RotateKeys(restore.recovery, account, c.builder.random, now)
	if err != nil {
		return identitylib.State{}, err
	}
	wrapped, ok := chain.GetWrappedFor(restore.device.SignerKey())
	if !ok {
		return identitylib.State{}, fmt.Errorf("%w: the rotation holds no keys for this device", identitylib.ErrUnwrap)
	}
	return identitylib.NewState(identitylib.Record{Device: restore.device, Wrapped: wrapped}, chain)
}

func (c restoreCompletion) openRecoveryKeys(recovery identitylib.RecoveryKey, chain identitylib.Chain) (identitylib.AccountKeys, error) {
	account, err := recovery.UnwrapAccountKeys(chain.WrappedForRecovery())
	if err != nil {
		return identitylib.AccountKeys{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidChain, err)
	}
	history, err := chain.OpenKeyHistory(account)
	if err != nil {
		return identitylib.AccountKeys{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidChain, err)
	}
	return history, nil
}
