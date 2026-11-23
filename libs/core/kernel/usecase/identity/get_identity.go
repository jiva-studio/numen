package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

type GetIdentity struct{ records records }

// NewGetIdentity returns the use case that reads the identity of this device.
func NewGetIdentity(d Deps) *GetIdentity { return &GetIdentity{records: d.records} }

// Execute returns the identity of this device, or identity.ErrNoIdentity when the key store holds no
// record. A record that does not parse, whose device is not in its chain, or that has no chain file, is
// reported as an error, never treated as absent.
func (u *GetIdentity) Execute(ctx context.Context) (identitydomain.Identity, error) {
	state, err := u.records.readState(ctx)
	if err != nil {
		return identitydomain.Identity{}, err
	}
	return toIdentity(state)
}
