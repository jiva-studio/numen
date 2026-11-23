package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

type GetPublicIdentity struct{ deps Deps }

// NewGetPublicIdentity returns the use case that reads what is public of the identity of this device.
func NewGetPublicIdentity(d Deps) *GetPublicIdentity { return &GetPublicIdentity{deps: d} }

// Execute returns the fingerprints, the public signing key of the device and the signed chain. It
// returns identity.ErrNoIdentity when the key store holds no record.
func (u *GetPublicIdentity) Execute(ctx context.Context) (identitydomain.Public, error) {
	record, err := u.deps.readRecord(ctx)
	if isNoKey(err) {
		return identitydomain.Public{}, identitydomain.ErrNoIdentity
	}
	if err != nil {
		return identitydomain.Public{}, err
	}
	shown, err := toIdentity(record)
	if err != nil {
		return identitydomain.Public{}, err
	}
	return identitydomain.Public{
		AccountFingerprint: shown.AccountFingerprint,
		DeviceFingerprint:  shown.DeviceFingerprint,
		DeviceSignKey:      record.Device.Public().Sign,
		Chain:              record.Chain.Bytes(),
	}, nil
}
