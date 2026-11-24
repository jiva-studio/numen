package identity

import (
	"context"
	"errors"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

type GetIdentity struct {
	records  records
	restores restoreHold
}

// NewGetIdentity returns the use case that reads the identity of this device.
func NewGetIdentity(d Deps) *GetIdentity {
	return &GetIdentity{records: d.records, restores: d.progress.restores}
}

// Execute returns the identity of this device, or identity.ErrNoIdentity when the key store holds no
// record. A record that does not parse, whose device is not in its chain, or that has no chain file, is
// reported as an error, never treated as absent. When this device holds none because a restore was refused, the
// error is the refusal of the restore (identity.ErrChainOutdated).
func (u *GetIdentity) Execute(ctx context.Context) (identitydomain.Identity, error) {
	state, err := u.records.readState(ctx)
	if err != nil {
		if refusal := u.restores.readRefusal(); refusal != nil && errors.Is(err, identitydomain.ErrNoIdentity) {
			return identitydomain.Identity{}, refusal
		}
		return identitydomain.Identity{}, err
	}
	return toIdentity(state)
}
