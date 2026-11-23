package identity

import (
	"context"
)

// StartRestoreFromKit starts to restore the account of two sheets of a kit on this device.
type StartRestoreFromKit struct{ deps Deps }

// NewStartRestoreFromKit returns the use case that starts a restore from sheets.
func NewStartRestoreFromKit(d Deps) *StartRestoreFromKit { return &StartRestoreFromKit{deps: d} }

// Execute restores the recovery secret from the sheets (ErrNotEnoughSheets, ErrInvalidSheets), refuses when an
// identity exists (ErrIdentityExists) and then keeps a pending restore exactly as StartRestore does for the words.
func (u *StartRestoreFromKit) Execute(ctx context.Context, sheets [][]string) error {
	secret, err := restoreSecret(sheets)
	if err != nil {
		return err
	}
	if err := checkAbsent(ctx, u.deps); err != nil {
		return err
	}
	recovery, err := secret.DeriveRecoveryKey()
	if err != nil {
		return err
	}
	return startPending(ctx, u.deps, recovery)
}
