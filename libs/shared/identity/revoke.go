package identity

import (
	"crypto/ed25519"
	"errors"
	"io"
	"time"
)

type Revoke struct {
	Device             ed25519.PublicKey
	Epoch              uint32
	AccountBox         [32]byte
	WrappedForRecovery WrappedAccountKeys
	PreviousKeys       []byte
	Wrapped            []DeviceWrap
}

type DeviceWrap struct {
	Device  ed25519.PublicKey
	Wrapped WrappedAccountKeys
}

func (c Chain) RevokeDevice(by Signer, current AccountKeys, device ed25519.PublicKey, random io.Reader, at time.Time) (Chain, AccountKeys, error) {
	return Chain{}, AccountKeys{}, errors.New("not implemented")
}

func (c Chain) Epoch() uint32 { return 0 }

func (c Chain) ListRevokedDevices() []Device { return nil }

func (c Chain) IsActiveDevice(sign ed25519.PublicKey) bool { return false }

func (c Chain) IsKnownDevice(sign ed25519.PublicKey) bool { return false }

func (c Chain) GetWrappedFor(device ed25519.PublicKey) (WrappedAccountKeys, bool) {
	return WrappedAccountKeys{}, false
}

func (c Chain) OpenKeyHistory(current AccountKeys) (AccountKeys, error) {
	return AccountKeys{}, errors.New("not implemented")
}

func (c Chain) RevokedSince(prefix Chain) []ed25519.PublicKey { return nil }

func (a AccountKeys) Epoch() uint32 { return 0 }
