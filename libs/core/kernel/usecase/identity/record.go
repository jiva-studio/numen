package identity

import (
	"context"
	"errors"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

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
