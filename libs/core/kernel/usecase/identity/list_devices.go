package identity

import (
	"bytes"
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

type ListDevices struct {
	records records
	links   linkHold
}

// NewListDevices returns the use case that lists the devices of the account.
func NewListDevices(d Deps) *ListDevices {
	return &ListDevices{records: d.records, links: d.progress.links}
}

// Execute returns the active devices of the chain of this device in the order they were added. A device whose revoke waits for the
// server is left out at once, and so are the revoked ones: a removed device disappears from the list.
// It returns ErrNoIdentity when this device holds none.
func (u *ListDevices) Execute(ctx context.Context) ([]identitydomain.Device, error) {
	state, err := u.records.readState(ctx)
	if err != nil {
		return nil, err
	}
	var removed []byte
	store, release := u.links.hold()
	if store.revoke != nil {
		removed = store.revoke.device
	}
	release()
	active := make([]identitylib.Device, 0)
	for _, device := range state.Chain.ListDevices() {
		if !bytes.Equal(device.Public.Sign, removed) {
			active = append(active, device)
		}
	}
	return listDevices(active, state.Record.Device.SignerKey(), false), nil
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
