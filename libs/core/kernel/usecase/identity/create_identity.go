package identity

import (
	"context"
	"encoding/base64"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// CreateIdentity keeps a new identity on this device.
type CreateIdentity struct{ deps Deps }

// NewCreateIdentity returns the use case that creates the identity of this device.
func NewCreateIdentity(d Deps) *CreateIdentity { return &CreateIdentity{deps: d} }

// Execute parses the words (ErrInvalidWords wrapping the library error), refuses when the key store
// holds an entry of the identity (ErrIdentityExists, even when it does not parse), derives the
// recovery key, makes new account keys and a device key, builds the chain (genesis and this device,
// signed by the recovery key), wraps the account keys for the device and writes the record in one
// SetKey. The words and the entropy are dropped when Execute returns.
func (u *CreateIdentity) Execute(ctx context.Context, words []string) (identitydomain.Identity, error) {
	recovery, err := identitylib.ParseRecoveryWords(words)
	if err != nil {
		return identitydomain.Identity{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidWords, err)
	}
	u.deps.createLock.Lock()
	defer u.deps.createLock.Unlock()
	if err := u.checkAbsent(ctx); err != nil {
		return identitydomain.Identity{}, err
	}
	record, err := u.buildRecord(recovery)
	if err != nil {
		return identitydomain.Identity{}, err
	}
	value := base64.RawURLEncoding.EncodeToString(record.Bytes())
	if err := u.deps.keys.SetKey(ctx, keyName, value); err != nil {
		return identitydomain.Identity{}, fmt.Errorf("%w: keep the identity: %w", identitydomain.ErrKeyStore, err)
	}
	return toIdentity(record)
}

func (u *CreateIdentity) checkAbsent(ctx context.Context) error {
	_, err := u.deps.keys.GetKey(ctx, keyName)
	switch {
	case err == nil:
		return identitydomain.ErrIdentityExists
	case isNoKey(err):
		return nil
	}
	return fmt.Errorf("%w: read the identity: %w", identitydomain.ErrKeyStore, err)
}

func (u *CreateIdentity) buildRecord(recovery identitylib.RecoveryWords) (identitylib.Record, error) {
	root, err := recovery.DeriveRecoveryKey()
	if err != nil {
		return identitylib.Record{}, err
	}
	account, err := identitylib.NewAccountKeys(u.deps.random)
	if err != nil {
		return identitylib.Record{}, err
	}
	device, err := identitylib.NewDeviceKey(u.deps.random)
	if err != nil {
		return identitylib.Record{}, err
	}
	now := u.deps.clock.Now()
	chain, err := identitylib.NewChain(root, account, now)
	if err != nil {
		return identitylib.Record{}, err
	}
	chain, err = chain.AppendDevice(root, device.Public(), normalizeEntryName(u.deps.names.GetInstanceName()), now)
	if err != nil {
		return identitylib.Record{}, err
	}
	wrapped, err := account.WrapForDevice(device.Public())
	if err != nil {
		return identitylib.Record{}, err
	}
	return identitylib.Record{Device: device, Wrapped: wrapped, Chain: chain}, nil
}
