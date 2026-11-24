package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

type GetIdentity struct{ deps Deps }

// NewGetIdentity returns the use case that reads the identity of this device.
func NewGetIdentity(d Deps) *GetIdentity { return &GetIdentity{deps: d} }

// Execute returns the identity of this device, or identity.ErrNoIdentity when the key store holds no
// record. A record that does not parse, or whose device is not in its chain, is reported as an error
// wrapping the ErrMalformed of the identity library, never treated as absent.
func (u *GetIdentity) Execute(ctx context.Context) (identitydomain.Identity, error) {
	record, err := u.deps.readRecord(ctx)
	if isNoKey(err) {
		return identitydomain.Identity{}, identitydomain.ErrNoIdentity
	}
	if err != nil {
		return identitydomain.Identity{}, err
	}
	return toIdentity(record)
}
