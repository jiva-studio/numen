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
	keys  func() pluginport.KeyStore
	names DeviceName
	clock machineport.Clock
}

// NewAccount returns the operations over the key store that keys returns at each call.
func NewAccount(keys func() pluginport.KeyStore, names DeviceName, clock machineport.Clock) *Account {
	return &Account{keys: keys, names: names, clock: clock}
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
	return NewDeps(store, a.names, a.clock), nil
}
