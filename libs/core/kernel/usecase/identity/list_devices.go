package identity

import (
	"bytes"
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

type ListDevices struct{ records records }

// NewListDevices returns the use case that lists the devices of the account.
func NewListDevices(d Deps) *ListDevices { return &ListDevices{records: d.records} }

// Execute returns the devices of the chain of this device: the active ones in the order they were added, then the revoked ones.
// It returns ErrNoIdentity when this device holds none.
func (u *ListDevices) Execute(ctx context.Context) ([]identitydomain.Device, error) {
	state, err := u.records.readState(ctx)
	if err != nil {
		return nil, err
	}
	own := state.Record.Device.SignerKey()
	devices := listDevices(state.Chain.ListDevices(), own, false)
	return append(devices, listDevices(state.Chain.ListRevokedDevices(), own, true)...), nil
}

func listDevices(devices []identitylib.Device, own []byte, isRevoked bool) []identitydomain.Device {
	listed := make([]identitydomain.Device, 0, len(devices))
	for _, device := range devices {
		listed = append(listed, identitydomain.Device{
			Name:        device.Name,
			Fingerprint: device.Public.Fingerprint().String(),
			AddedAt:     device.AddedAt,
			IsThis:      bytes.Equal(device.Public.Sign, own),
			IsRevoked:   isRevoked,
		})
	}
	return listed
}
