package identity

import (
	"context"
	"errors"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

var errProof = errors.New("give the recovery words or sheets, not both and not none")

// CreateRecoveryKit makes a new kit of three sheets for the account of this device.
type CreateRecoveryKit struct{ deps Deps }

// NewCreateRecoveryKit returns the use case that makes a recovery kit.
func NewCreateRecoveryKit(d Deps) *CreateRecoveryKit { return &CreateRecoveryKit{deps: d} }

// Execute proves the account first with exactly one of words and sheets, then makes the kit. The
// proof must belong to the account of this device (ErrOtherAccount). Nothing is kept: no word, no
// secret and no sheet is written.
func (u *CreateRecoveryKit) Execute(ctx context.Context, words []string, sheets [][]string) (identitydomain.Kit, error) {
	secret, err := proveSecret(words, sheets)
	if err != nil {
		return identitydomain.Kit{}, err
	}
	account, err := u.deps.proveAccount(ctx, secret)
	if err != nil {
		return identitydomain.Kit{}, err
	}
	made, err := identitylib.NewKit(u.deps.random, secret)
	if err != nil {
		return identitydomain.Kit{}, err
	}
	return writeKit(made, account), nil
}

func proveSecret(words []string, sheets [][]string) (identitylib.RecoverySecret, error) {
	if (len(words) > 0) == (len(sheets) > 0) {
		return identitylib.RecoverySecret{}, errProof
	}
	if len(sheets) > 0 {
		return restoreSecret(sheets)
	}
	recovery, err := identitylib.ParseRecoveryWords(words)
	if err != nil {
		return identitylib.RecoverySecret{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidWords, err)
	}
	return recovery.DeriveRecoverySecret(), nil
}

func writeKit(made identitylib.Kit, account string) identitydomain.Kit {
	sheets := make([]identitydomain.Sheet, len(made.Sheets))
	for i, sheet := range made.Sheets {
		sheets[i] = identitydomain.Sheet{Number: sheet.Number(), Words: sheet.Words()}
	}
	return identitydomain.Kit{ID: made.ID, AccountFingerprint: account, Sheets: sheets}
}
