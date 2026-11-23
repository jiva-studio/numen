package identity

import (
	"context"
	"fmt"
	"io"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// CreateIdentity keeps a new identity on this device.
type CreateIdentity struct {
	records records
	builder stateBuilder
}

// NewCreateIdentity returns the use case that creates the identity of this device.
func NewCreateIdentity(d Deps) *CreateIdentity {
	return &CreateIdentity{records: d.records, builder: d.getBuilder()}
}

// Execute parses the words (ErrInvalidWords wrapping the library error), refuses when the key store
// holds an entry of the identity (ErrIdentityExists, even when it does not parse), derives the
// recovery key, makes new account keys and a device key, builds the chain (genesis and this device,
// signed by the recovery key), wraps the account keys for the device, writes the chain file and then
// the record with one SetKey. The words and the entropy are dropped when Execute returns.
func (u *CreateIdentity) Execute(ctx context.Context, words []string) (identitydomain.Identity, error) {
	recovery, err := identitylib.ParseRecoveryWords(words)
	if err != nil {
		return identitydomain.Identity{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidWords, err)
	}
	if err := u.records.checkAbsent(ctx); err != nil {
		return identitydomain.Identity{}, err
	}
	state, err := u.buildState(recovery)
	if err != nil {
		return identitydomain.Identity{}, err
	}
	if err := u.records.writeNewState(ctx, state.Record, state.Chain); err != nil {
		return identitydomain.Identity{}, err
	}
	return identityOf(state)
}

func (u *CreateIdentity) buildState(recovery identitylib.RecoveryWords) (identitylib.State, error) {
	root, err := recovery.DeriveRecoveryKey()
	if err != nil {
		return identitylib.State{}, err
	}
	device, err := identitylib.NewDeviceKey(u.builder.random)
	if err != nil {
		return identitylib.State{}, err
	}
	return u.builder.buildNew(root, device)
}

// stateBuilder makes the state of a device that starts an account, or that joins one with its recovery words.
type stateBuilder struct {
	names  DeviceName
	clock  Clock
	random io.Reader
}

func (b stateBuilder) buildNew(root identitylib.RecoveryKey, device identitylib.DeviceKey) (identitylib.State, error) {
	account, err := identitylib.NewAccountKeys(b.random)
	if err != nil {
		return identitylib.State{}, err
	}
	now := b.clock.Now()
	chain, err := identitylib.NewChain(root, account, now)
	if err != nil {
		return identitylib.State{}, err
	}
	chain, err = chain.AppendDevice(root, device.Public(), entryName(b.names.GetInstanceName()), now)
	if err != nil {
		return identitylib.State{}, err
	}
	wrapped, err := account.WrapForDevice(device.Public())
	if err != nil {
		return identitylib.State{}, err
	}
	return identitylib.NewState(identitylib.Record{Device: device, Wrapped: wrapped}, chain)
}
