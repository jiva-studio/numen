package identity

import (
	"bytes"
	"context"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// CompleteRestore finishes the pending restore with the chain a sync plugin read from its server.
type CompleteRestore struct{ completion restoreCompletion }

// NewCompleteRestore returns the use case that completes a restore.
func NewCompleteRestore(d Deps) *CompleteRestore {
	return &CompleteRestore{completion: d.getCompletion()}
}

// Execute joins the account of chainBytes: the chain must verify and have the root of the pending words, the
// recovery key unwraps the account keys, and this device is appended and wrapped for. Empty chainBytes means the
// server has no such account, and the result is a new identity of the words. The record is written with one
// SetKey and the pending restore is cleared. ErrNoPendingRestore, ErrInvalidChain and ErrIdentityExists refuse it.
func (u *CompleteRestore) Execute(ctx context.Context, pluginID string, chainBytes []byte) error {
	if pluginID == "" {
		return errNoPluginID
	}
	restore := u.completion.restores.read()
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
}

func (c restoreCompletion) complete(ctx context.Context, restore *pendingRestore, chainBytes []byte) error {
	store, release := c.restores.hold()
	defer release()
	if store.current != restore {
		return identitydomain.ErrNoPendingRestore
	}
	if err := c.records.checkAbsent(ctx); err != nil {
		return err
	}
	state, err := c.buildRestoredState(restore, chainBytes)
	if err != nil {
		return err
	}
	if err := c.records.writeNewState(ctx, state.Record, state.Chain); err != nil {
		return err
	}
	store.clearLocked()
	return nil
}

func (c restoreCompletion) buildRestoredState(restore *pendingRestore, chainBytes []byte) (identitylib.State, error) {
	if len(chainBytes) == 0 {
		return c.builder.buildNew(restore.recovery, restore.device)
	}
	chain, err := identitylib.ParseChain(chainBytes)
	if err != nil {
		return identitylib.State{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidChain, err)
	}
	if !bytes.Equal(chain.Root().Sign, restore.recovery.Public().Sign) {
		return identitylib.State{}, fmt.Errorf("%w: the chain has another root", identitydomain.ErrInvalidChain)
	}
	account, err := restore.recovery.UnwrapAccountKeys(chain.WrappedForRecovery())
	if err != nil {
		return identitylib.State{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidChain, err)
	}
	name := normalizeEntryName(c.builder.names.GetInstanceName())
	chain, err = chain.AppendDevice(restore.recovery, restore.device.Public(), name, c.builder.clock.Now())
	if err != nil {
		return identitylib.State{}, err
	}
	wrapped, err := account.WrapForDevice(restore.device.Public())
	if err != nil {
		return identitylib.State{}, err
	}
	return identitylib.NewState(identitylib.Record{Device: restore.device, Wrapped: wrapped}, chain)
}
