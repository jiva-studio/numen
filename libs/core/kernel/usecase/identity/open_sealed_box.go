package identity

import (
	"context"
	"errors"
)

// OpenSealedBox opens a box that was sealed to the account box key.
type OpenSealedBox struct{ deps Deps }

// NewOpenSealedBox returns the use case that opens a box sealed to the account for a plugin.
func NewOpenSealedBox(d Deps) *OpenSealedBox { return &OpenSealedBox{deps: d} }

// Execute opens sealed for the plugin and the context. Bytes that do not open are ErrCannotOpen; a device with no
// record is identity.ErrNoIdentity.
func (u *OpenSealedBox) Execute(_ context.Context, _ string, _, _ []byte) ([]byte, error) {
	return nil, errors.New("not implemented")
}
