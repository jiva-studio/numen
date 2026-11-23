package identity

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func (d Deps) createFromRecoveryKey(ctx context.Context, root identitylib.RecoveryKey) (identitydomain.Identity, error) {
	d.writes.Lock()
	defer d.writes.Unlock()
	if err := d.checkAbsent(ctx); err != nil {
		return identitydomain.Identity{}, err
	}
	record, err := d.buildRecord(root)
	if err != nil {
		return identitydomain.Identity{}, err
	}
	value := base64.RawURLEncoding.EncodeToString(record.Bytes())
	if err := d.keys.SetKey(ctx, keyName, value); err != nil {
		return identitydomain.Identity{}, fmt.Errorf("%w: keep the identity: %w", identitydomain.ErrKeyStore, err)
	}
	return toIdentity(record)
}

func (d Deps) checkAbsent(ctx context.Context) error {
	_, err := d.keys.GetKey(ctx, keyName)
	switch {
	case err == nil:
		return identitydomain.ErrIdentityExists
	case isNoKey(err):
		return nil
	}
	return fmt.Errorf("%w: read the identity: %w", identitydomain.ErrKeyStore, err)
}

func (d Deps) buildRecord(root identitylib.RecoveryKey) (identitylib.Record, error) {
	account, err := identitylib.NewAccountKeys(d.random)
	if err != nil {
		return identitylib.Record{}, err
	}
	device, err := identitylib.NewDeviceKey(d.random)
	if err != nil {
		return identitylib.Record{}, err
	}
	now := d.clock.Now()
	chain, err := identitylib.NewChain(root, account, now)
	if err != nil {
		return identitylib.Record{}, err
	}
	chain, err = chain.AppendDevice(root, device.Public(), normalizeEntryName(d.names.GetInstanceName()), now)
	if err != nil {
		return identitylib.Record{}, err
	}
	wrapped, err := account.WrapForDevice(device.Public())
	if err != nil {
		return identitylib.Record{}, err
	}
	return identitylib.Record{Device: device, Wrapped: wrapped, Chain: chain}, nil
}

func (d Deps) proveAccount(ctx context.Context, secret identitylib.RecoverySecret) (string, error) {
	record, err := d.readRecord(ctx)
	if isNoKey(err) {
		return "", identitydomain.ErrNoIdentity
	}
	if err != nil {
		return "", err
	}
	key, err := secret.DeriveRecoveryKey()
	if err != nil {
		return "", err
	}
	fingerprint := key.Public().Fingerprint()
	if fingerprint != record.Chain.Root().Fingerprint() {
		return "", identitydomain.ErrOtherAccount
	}
	return fingerprint.String(), nil
}

func restoreSecret(sheets [][]string) (identitylib.RecoverySecret, error) {
	parsed := make([]identitylib.Sheet, len(sheets))
	for i, words := range sheets {
		sheet, err := identitylib.ParseSheet(words)
		if err != nil {
			return identitylib.RecoverySecret{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidSheets, &identitylib.SheetError{Position: i + 1, Err: err})
		}
		parsed[i] = sheet
	}
	secret, err := identitylib.RestoreSecret(parsed)
	switch {
	case errors.Is(err, identitylib.ErrNotEnoughSheets):
		return identitylib.RecoverySecret{}, fmt.Errorf("%w: %w", identitydomain.ErrNotEnoughSheets, err)
	case err != nil:
		return identitylib.RecoverySecret{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidSheets, err)
	}
	return secret, nil
}
