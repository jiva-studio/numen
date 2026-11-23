package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

type GetPublicIdentity struct{ records records }

// NewGetPublicIdentity returns the use case that reads what is public of the identity of this device.
func NewGetPublicIdentity(d Deps) *GetPublicIdentity { return &GetPublicIdentity{records: d.records} }

// Execute returns the fingerprints, the public signing key of the device, the signed chain, the epoch of the
// keys, the length of the chain and whether the account revoked this device. It returns identity.ErrNoIdentity
// when the key store holds no record.
func (u *GetPublicIdentity) Execute(ctx context.Context) (identitydomain.Public, error) {
	state, err := u.records.readState(ctx)
	if err != nil {
		return identitydomain.Public{}, err
	}
	shown, err := identityOf(state)
	if err != nil {
		return identitydomain.Public{}, err
	}
	return identitydomain.Public{
		AccountFingerprint: shown.AccountFingerprint,
		DeviceFingerprint:  shown.DeviceFingerprint,
		DeviceSignKey:      state.Record.Device.Public().Sign,
		Chain:              state.Chain.Bytes(),
		KeyEpoch:           shown.KeyEpoch,
		ChainLength:        state.Chain.Len(),
		IsRevoked:          state.Status == identitylib.StatusRevoked,
	}, nil
}
