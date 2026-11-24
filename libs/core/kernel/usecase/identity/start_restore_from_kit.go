package identity

import (
	"context"
)

// StartRestoreFromKit starts to restore the account of two sheets of a kit on this device.
type StartRestoreFromKit struct{ completion restoreCompletion }

// NewStartRestoreFromKit returns the use case that starts a restore from sheets.
func NewStartRestoreFromKit(d Deps) *StartRestoreFromKit {
	return &StartRestoreFromKit{completion: d.getCompletion()}
}

// Execute restores the recovery secret from the sheets (ErrNotEnoughSheets, ErrInvalidSheets), refuses when an
// identity exists (ErrIdentityExists) and then keeps a pending restore exactly as StartRestore does for the words. The mark is the chain mark printed on the kit.
func (u *StartRestoreFromKit) Execute(ctx context.Context, sheets [][]string, mark string) error {
	secret, err := restoreSecret(sheets)
	if err != nil {
		return err
	}
	if err := u.completion.records.checkAbsent(ctx); err != nil {
		return err
	}
	recovery, err := secret.DeriveRecoveryKey()
	if err != nil {
		return err
	}
	return u.completion.start(ctx, recovery, mark)
}
