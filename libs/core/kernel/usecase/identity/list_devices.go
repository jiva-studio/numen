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

// Execute returns the devices of the chain of this device: the active ones in the order they were added, then the revoked ones, each with
// when, by whom and why it was revoked. It returns ErrNoIdentity when this device holds none.
func (u *ListDevices) Execute(ctx context.Context) ([]identitydomain.Device, error) {
	state, err := u.records.readState(ctx)
	if err != nil {
		return nil, err
	}
	own := state.Record.Device.SignerKey()
	active, revoked := state.Chain.ListDevices(), state.Chain.ListRevokedDevices()
	everyone := append(append([]identitylib.Device{}, active...), revoked...)
	root := state.Chain.Root().Sign
	devices := listDevices(active, own, everyone, root)
	return append(devices, listDevices(revoked, own, everyone, root)...), nil
}

func listDevices(devices []identitylib.Device, own []byte, everyone []identitylib.Device, root []byte) []identitydomain.Device {
	listed := make([]identitydomain.Device, 0, len(devices))
	for _, device := range devices {
		listed = append(listed, identitydomain.Device{
			Name:        device.Name,
			Fingerprint: device.Public.Fingerprint().String(),
			AddedAt:     device.AddedAt,
			IsThis:      bytes.Equal(device.Public.Sign, own),
			IsRevoked:   len(device.Removal.By) > 0,
			Revocation:  readRevocation(device.Removal, everyone, root),
		})
	}
	return listed
}

func readRevocation(removal identitylib.Removal, everyone []identitylib.Device, root []byte) identitydomain.Revocation {
	if len(removal.By) == 0 {
		return identitydomain.Revocation{}
	}
	revocation := identitydomain.Revocation{At: removal.At, Reason: identitydomain.RevokeLost, Since: removal.Since, IsByRecovery: bytes.Equal(removal.By, root)}
	if removal.Reason == identitylib.ReasonStolen {
		revocation.Reason = identitydomain.RevokeStolen
	}
	for _, device := range everyone {
		if bytes.Equal(device.Public.Sign, removal.By) {
			revocation.ByName = device.Name
		}
	}
	return revocation
}
