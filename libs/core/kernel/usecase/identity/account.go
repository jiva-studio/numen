package identity

import (
	"context"
	"errors"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

var errNoKeyStore = errors.New("this installation has no key store")

// Account is what a plugin with the identity permission may ask of the identity of this device. It reads
// the key store when a call is made, so a key store set after the plugins start is the one used.
type Account struct {
	keys    func() pluginport.KeyStore
	names   DeviceName
	clock   Clock
	pending *PendingStore
}

// NewAccount returns the operations over the key store that keys returns at each call.
func NewAccount(keys func() pluginport.KeyStore, names DeviceName, clock Clock, pending *PendingStore) *Account {
	if pending == nil {
		pending = NewPendingStore()
	}
	return &Account{keys: keys, names: names, clock: clock, pending: pending}
}

// SignAsDevice signs message with the device key under the plugin id.
func (a *Account) SignAsDevice(ctx context.Context, pluginID string, message []byte) ([]byte, error) {
	deps, err := a.open()
	if err != nil {
		return nil, err
	}
	return NewSignAsDevice(deps).Execute(ctx, pluginID, message)
}

// GetPublicIdentity returns what is public of the identity of this device.
func (a *Account) GetPublicIdentity(ctx context.Context) (identitydomain.Public, error) {
	deps, err := a.open()
	if err != nil {
		return identitydomain.Public{}, err
	}
	return NewGetPublicIdentity(deps).Execute(ctx)
}

func (a *Account) open() (Deps, error) {
	store := a.keys()
	if store == nil {
		return Deps{}, errNoKeyStore
	}
	return NewDeps(store, a.names, a.clock).WithPending(a.pending), nil
}

// SealData seals plain for the plugin with the account keys of this device.
func (a *Account) SealData(ctx context.Context, pluginID string, dataContext, plain []byte) ([]byte, error) {
	deps, err := a.open()
	if err != nil {
		return nil, err
	}
	return NewSealData(deps).Execute(ctx, pluginID, dataContext, plain)
}

// OpenData fails with ErrCannotOpen for a value sealed for another plugin or context, or with other keys.
func (a *Account) OpenData(ctx context.Context, pluginID string, dataContext, sealed []byte) ([]byte, error) {
	deps, err := a.open()
	if err != nil {
		return nil, err
	}
	return NewOpenData(deps).Execute(ctx, pluginID, dataContext, sealed)
}

// GetPendingRestore returns the public keys of the pending restore.
func (a *Account) GetPendingRestore(ctx context.Context) (identitydomain.PendingRestore, error) {
	deps, err := a.open()
	if err != nil {
		return identitydomain.PendingRestore{}, err
	}
	return NewGetPendingRestore(deps).Execute(ctx)
}

// SignAsRecovery signs the message with the recovery key of the pending restore.
func (a *Account) SignAsRecovery(ctx context.Context, pluginID string, message []byte) ([]byte, error) {
	deps, err := a.open()
	if err != nil {
		return nil, err
	}
	return NewSignAsRecovery(deps).Execute(ctx, pluginID, message)
}

// CompleteRestore completes the pending restore with the chain of the server, or with none.
func (a *Account) CompleteRestore(ctx context.Context, pluginID string, chain []byte) error {
	deps, err := a.open()
	if err != nil {
		return err
	}
	return NewCompleteRestore(deps).Execute(ctx, pluginID, chain)
}
