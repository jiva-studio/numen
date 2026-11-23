package identity

import (
	"bytes"
	"context"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// RevokeDevice removes a device from the account.
type RevokeDevice struct{ deps Deps }

// NewRevokeDevice returns the use case that revokes a device.
func NewRevokeDevice(d Deps) *RevokeDevice { return &RevokeDevice{deps: d} }

// Execute makes the chain that revokes the device with the fingerprint and the keys of the next epoch, and keeps them in memory until the
// server accepted the chain. It never revokes this device (identitylib.ErrSelfRevoke) and never leaves the account with no device
// (ErrLastDevice); a device that is not active is identitylib.ErrUnknownDevice.
func (u *RevokeDevice) Execute(ctx context.Context, fingerprint string) error {
	state, err := u.deps.readState(ctx)
	if err != nil {
		return err
	}
	if state.Status == identitylib.StatusRevoked {
		return identitydomain.ErrDeviceRevoked
	}
	target, err := findActiveDevice(state.Chain, state.Record.Device.SignerKey(), fingerprint)
	if err != nil {
		return err
	}
	chain, err := u.makeChain(state, target)
	if err != nil {
		return err
	}
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	store.revoke = &pendingRevoke{chain: chain, device: target}
	store.bumpLocked()
	return nil
}

func findActiveDevice(chain identitylib.Chain, own []byte, fingerprint string) ([]byte, error) {
	active := chain.ListDevices()
	if len(active) <= 1 {
		return nil, identitydomain.ErrLastDevice
	}
	for _, device := range active {
		if device.Public.Fingerprint().String() != fingerprint {
			continue
		}
		if bytes.Equal(device.Public.Sign, own) {
			return nil, identitylib.ErrSelfRevoke
		}
		return device.Public.Sign, nil
	}
	return nil, fmt.Errorf("%w: no active device has the fingerprint", identitylib.ErrUnknownDevice)
}

func (u *RevokeDevice) makeChain(state identitylib.State, target []byte) (identitylib.Chain, error) {
	keys, err := state.OpenAccountKeys()
	if err != nil {
		return identitylib.Chain{}, err
	}
	chain, _, err := state.Chain.RevokeDevice(state.Record.Device, keys, target, u.deps.random, u.deps.clock.Now())
	return chain, err
}
