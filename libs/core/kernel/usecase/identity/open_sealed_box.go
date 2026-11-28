package identity

import (
	"context"
	"errors"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// OpenSealedBox opens a box that was sealed to the account box key.
type OpenSealedBox struct{ records records }

// NewOpenSealedBox returns the use case that opens a box sealed to the account for a plugin.
func NewOpenSealedBox(d Deps) *OpenSealedBox { return &OpenSealedBox{records: d.records} }

// Execute opens sealed for the plugin and the context. Bytes that do not open are ErrCannotOpen; a device with no
// record is identity.ErrNoIdentity.
func (u *OpenSealedBox) Execute(ctx context.Context, pluginID string, boxContext, sealed []byte) ([]byte, error) {
	keys, err := readAccountKeys(ctx, u.records, pluginID)
	if err != nil {
		return nil, err
	}
	plain, err := keys.OpenSealedBox(pluginID, boxContext, sealed)
	if errors.Is(err, identitylib.ErrUnwrap) {
		return nil, fmt.Errorf("%w: %w", identitydomain.ErrCannotOpen, err)
	}
	return plain, err
}
