package identity

import (
	"context"
	"errors"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

var errNoKeyStore = errors.New("this installation has no key store")

// Account is what a plugin with the identity permission may ask of the identity of this device. It reads
// the key store when a call is made, so a key store set after the plugins start is the one used.
type Account struct {
	keys     func() pluginport.KeyStore
	names    DeviceName
	clock    machineport.InstallationClock
	chains   ChainStore
	progress *Progress
}

// NewAccount returns the operations over the key store that keys returns at each call, the chain file and what is in progress.
func NewAccount(keys func() pluginport.KeyStore, names DeviceName, clock machineport.InstallationClock, progress *Progress, chains ChainStore) *Account {
	if progress == nil {
		progress = NewProgress()
	}
	return &Account{keys: keys, names: names, clock: clock, chains: chains, progress: progress}
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
	if store == nil || a.chains == nil {
		return Deps{}, errNoKeyStore
	}
	return NewDeps(store, a.names, a.clock).WithProgress(a.progress).WithChains(a.chains), nil
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

// OpenSealedBox opens a box sealed to the account box key for the plugin and the context. It fails with
// ErrCannotOpen for a box sealed for another plugin, context or account.
func (a *Account) OpenSealedBox(ctx context.Context, pluginID string, boxContext, sealed []byte) ([]byte, error) {
	deps, err := a.open()
	if err != nil {
		return nil, err
	}
	return NewOpenSealedBox(deps).Execute(ctx, pluginID, boxContext, sealed)
}
