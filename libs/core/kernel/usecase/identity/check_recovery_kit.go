package identity

import "context"

// CheckRecoveryKit reports whether two sheets restore the account of this device.
type CheckRecoveryKit struct{ deps Deps }

// NewCheckRecoveryKit returns the use case that checks sheets against this device.
func NewCheckRecoveryKit(d Deps) *CheckRecoveryKit { return &CheckRecoveryKit{deps: d} }

// Execute restores the secret from the sheets and compares its account with the record of this
// device. It returns the account fingerprint, and keeps nothing.
func (u *CheckRecoveryKit) Execute(ctx context.Context, sheets [][]string) (string, error) {
	secret, err := restoreSecret(sheets)
	if err != nil {
		return "", err
	}
	return u.deps.proveAccount(ctx, secret)
}
