package identity

import (
	"errors"
	"io"
)

// SealToAccountBox seals plain to the account box key of a person. Anyone may call it.
func SealToAccountBox(_ io.Reader, _ [32]byte, _ string, _, _ []byte) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// OpenSealedBox opens a box sealed to the box key of these account keys, for that plugin and context.
// Any failure is ErrUnwrap.
func (a AccountKeys) OpenSealedBox(_ string, _, _ []byte) ([]byte, error) {
	return nil, errors.New("not implemented")
}
