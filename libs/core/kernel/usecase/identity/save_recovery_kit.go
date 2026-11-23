package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

var errProof = errors.New("give the recovery words or sheets, not both and not none")

// KitSaver writes a recovery kit as a document to the file the person chooses.
type KitSaver interface {
	// SaveKit asks the person where to save the kit and writes it there. It returns false when the
	// person chose no file.
	SaveKit(ctx context.Context, kit identitydomain.Kit) (bool, error)
}

// SaveRecoveryKit makes a new kit of three sheets for the account of this device and saves it.
type SaveRecoveryKit struct {
	deps  Deps
	saver KitSaver
}

// NewSaveRecoveryKit returns the use case that makes a recovery kit and saves it through saver.
func NewSaveRecoveryKit(d Deps, saver KitSaver) *SaveRecoveryKit {
	return &SaveRecoveryKit{deps: d, saver: saver}
}

// Execute proves the account first with exactly one of words and sheets, then makes the kit and
// hands it to the saver. The proof must belong to the account of this device (ErrOtherAccount). The
// key store is not written: no word, no secret and no sheet is kept. It returns false when the
// person chose no file.
func (u *SaveRecoveryKit) Execute(ctx context.Context, words []string, sheets [][]string) (bool, error) {
	secret, err := proveSecret(words, sheets)
	if err != nil {
		return false, err
	}
	account, err := u.deps.proveAccount(ctx, secret)
	if err != nil {
		return false, err
	}
	made, err := identitylib.NewKit(u.deps.random, secret)
	if err != nil {
		return false, err
	}
	return u.saver.SaveKit(ctx, writeKit(made, account, u.deps.clock.Now()))
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

func writeKit(made identitylib.Kit, account string, now time.Time) identitydomain.Kit {
	sheets := make([]identitydomain.Sheet, len(made.Sheets))
	for i, sheet := range made.Sheets {
		sheets[i] = identitydomain.Sheet{Number: sheet.Number(), Words: sheet.Words()}
	}
	return identitydomain.Kit{ID: made.ID, AccountFingerprint: account, CreatedAt: now, Sheets: sheets}
}
