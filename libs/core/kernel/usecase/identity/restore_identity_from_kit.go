package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// RestoreIdentityFromKit keeps an identity on this device from two sheets of a kit.
type RestoreIdentityFromKit struct{ deps Deps }

// NewRestoreIdentityFromKit returns the use case that restores the identity from sheets.
func NewRestoreIdentityFromKit(d Deps) *RestoreIdentityFromKit {
	return &RestoreIdentityFromKit{deps: d}
}

// Execute restores the recovery secret from the sheets and creates the identity like CreateIdentity:
// it never overwrites an entry of the identity (ErrIdentityExists), makes new account keys and a
// device key and writes one record. The account fingerprint is the one of the words.
func (u *RestoreIdentityFromKit) Execute(ctx context.Context, sheets [][]string) (identitydomain.Identity, error) {
	secret, err := restoreSecret(sheets)
	if err != nil {
		return identitydomain.Identity{}, err
	}
	root, err := secret.DeriveRecoveryKey()
	if err != nil {
		return identitydomain.Identity{}, err
	}
	return u.deps.createFromRecoveryKey(ctx, root)
}
