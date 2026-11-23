package identity

import (
	"bytes"
	"context"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// CompleteRestore finishes the pending restore with the chain a sync plugin read from its server.
type CompleteRestore struct{ deps Deps }

// NewCompleteRestore returns the use case that completes a restore.
func NewCompleteRestore(d Deps) *CompleteRestore { return &CompleteRestore{deps: d} }

// Execute joins the account of chainBytes: the chain must verify and have the root of the pending words, the
// recovery key unwraps the account keys, and this device is appended and wrapped for. Empty chainBytes means the
// server has no such account, and the result is a new identity of the words. The record is written with one
// SetKey and the pending restore is cleared. ErrNoPendingRestore, ErrInvalidChain and ErrIdentityExists refuse it.
func (u *CompleteRestore) Execute(ctx context.Context, pluginID string, chainBytes []byte) error {
	if pluginID == "" {
		return errNoPluginID
	}
	restore := u.deps.pending.read()
	if restore == nil {
		return identitydomain.ErrNoPendingRestore
	}
	return completeRestore(ctx, u.deps, restore, chainBytes)
}

func completeRestore(ctx context.Context, deps Deps, restore *pendingRestore, chainBytes []byte) error {
	deps.pending.mu.Lock()
	defer deps.pending.mu.Unlock()
	if deps.pending.current != restore {
		return identitydomain.ErrNoPendingRestore
	}
	if err := checkAbsent(ctx, deps); err != nil {
		return err
	}
	state, err := buildRestoredState(deps, restore, chainBytes)
	if err != nil {
		return err
	}
	if err := deps.writeNewState(ctx, state.Record, state.Chain); err != nil {
		return err
	}
	deps.pending.clearLocked()
	return nil
}

func buildRestoredState(deps Deps, restore *pendingRestore, chainBytes []byte) (identitylib.State, error) {
	if len(chainBytes) == 0 {
		return buildNewState(deps, restore.recovery, restore.device)
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
	chain, err = chain.AppendDevice(restore.recovery, restore.device.Public(), entryName(deps.names.GetInstanceName()), deps.clock.Now())
	if err != nil {
		return identitylib.State{}, err
	}
	wrapped, err := account.WrapForDevice(restore.device.Public())
	if err != nil {
		return identitylib.State{}, err
	}
	return identitylib.NewState(identitylib.Record{Device: restore.device, Wrapped: wrapped}, chain)
}
