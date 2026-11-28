package identity

import (
	"context"
	"errors"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

type GetIdentity struct {
	records  records
	restores restoreReader
}

// NewGetIdentity returns the use case that reads the identity of this device.
func NewGetIdentity(d Deps) *GetIdentity {
	return &GetIdentity{records: d.records, restores: d.progress.pending}
}

// Execute returns the identity of this device, or identity.ErrNoIdentity when the key store holds no
// record, no chain file, a record that does not parse or a device that is not in its chain: none of them can be used, and a new
// identity, a restore or a link replaces it. When this device holds none because a restore was refused, the
// error is the refusal of the restore (identity.ErrChainOutdated).
func (u *GetIdentity) Execute(ctx context.Context) (identitydomain.Identity, error) {
	state, err := u.records.readState(ctx)
	if err != nil {
		if refusal := u.restores.readRefusal(); refusal != nil && errors.Is(err, identitydomain.ErrNoIdentity) {
			return identitydomain.Identity{}, refusal
		}
		return identitydomain.Identity{}, err
	}
	found, err := toIdentity(state)
	if errors.Is(err, identitylib.ErrMalformed) {
		return identitydomain.Identity{}, fmt.Errorf("%w: %w", identitydomain.ErrNoIdentity, err)
	}
	return found, err
}
