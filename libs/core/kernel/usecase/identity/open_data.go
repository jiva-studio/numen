package identity

import (
	"context"
	"errors"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// OpenData decrypts data that SealData sealed.
type OpenData struct{ records records }

// NewOpenData returns the use case that opens data sealed for a plugin.
func NewOpenData(d Deps) *OpenData { return &OpenData{records: d.records} }

// Execute opens sealed for the plugin and the context. Bytes that do not open are ErrCannotOpen; a device with no
// record is identity.ErrNoIdentity.
func (u *OpenData) Execute(ctx context.Context, pluginID string, dataContext, sealed []byte) ([]byte, error) {
	keys, err := readAccountKeys(ctx, u.records, pluginID)
	if err != nil {
		return nil, err
	}
	plain, err := keys.OpenForPlugin(pluginID, dataContext, sealed)
	if errors.Is(err, identitylib.ErrUnwrap) {
		return nil, fmt.Errorf("%w: %w", identitydomain.ErrCannotOpen, err)
	}
	return plain, err
}
