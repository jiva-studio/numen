package identity

import (
	"context"
	"errors"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

var errNoPluginID = errors.New("a signature needs the id of the plugin")

type SignAsDevice struct{ deps Deps }

// NewSignAsDevice returns the use case that signs a message of a plugin with the device key.
func NewSignAsDevice(d Deps) *SignAsDevice { return &SignAsDevice{deps: d} }

// Execute signs message under the plugin id, so a signature made for one plugin never verifies for
// another. It returns identity.ErrNoIdentity when the key store holds no record.
func (u *SignAsDevice) Execute(ctx context.Context, pluginID string, message []byte) ([]byte, error) {
	if pluginID == "" {
		return nil, errNoPluginID
	}
	record, err := u.deps.readRecord(ctx)
	if isNoKey(err) {
		return nil, identitydomain.ErrNoIdentity
	}
	if err != nil {
		return nil, err
	}
	signature := record.Device.SignForPlugin(pluginID, message)
	return signature[:], nil
}
