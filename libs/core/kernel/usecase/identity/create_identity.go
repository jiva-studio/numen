package identity

import (
	"context"
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
	root, err := recovery.DeriveRecoveryKey()
	if err != nil {
		return identitydomain.Identity{}, err
	}
	return u.deps.createFromRecoveryKey(ctx, root)
}
