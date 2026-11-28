package identity

import (
	"context"
	"errors"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func (r records) proveAccount(ctx context.Context, secret identitylib.RecoverySecret) (string, error) {
	state, err := r.readState(ctx)
	if err != nil {
		return "", err
	}
	key, err := secret.DeriveRecoveryKey()
	if err != nil {
		return "", err
	}
	fingerprint := key.Public().Fingerprint()
	if fingerprint != state.Chain.Root().Fingerprint() {
		return "", identitydomain.ErrOtherAccount
	}
	return fingerprint.String(), nil
}

func restoreSecret(sheets [][]string) (identitylib.RecoverySecret, error) {
	parsed := make([]identitylib.Sheet, len(sheets))
	for i, words := range sheets {
		sheet, err := identitylib.ParseSheet(words)
		if err != nil {
			return identitylib.RecoverySecret{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidSheets, &identitydomain.SheetError{Position: i + 1, IsDuplicate: false, Err: err})
		}
		parsed[i] = sheet
	}
	secret, err := identitylib.RestoreSecret(parsed)
	switch {
	case errors.Is(err, identitylib.ErrNotEnoughSheets):
		return identitylib.RecoverySecret{}, fmt.Errorf("%w: %w", identitydomain.ErrNotEnoughSheets, err)
	case err != nil:
		return identitylib.RecoverySecret{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidSheets, mapSheetError(err))
	}
	return secret, nil
}
