package identityport

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// Operations is what a plugin with the identity permission may ask of the identity of this device.
// Both return identity.ErrNoIdentity when the device holds none.
type Operations interface {
	// SignAsDevice signs message with the device key under the plugin id.
	SignAsDevice(ctx context.Context, pluginID string, message []byte) ([]byte, error)
	// GetPublicIdentity returns what is public of the identity of this device.
	GetPublicIdentity(ctx context.Context) (identitydomain.Public, error)
}
