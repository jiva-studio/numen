// Package digest is the running SHA-256 hash of the stored bytes of a blob.
package digest

import (
	"crypto/sha256"
	"encoding"
	"errors"
	"fmt"
	"hash"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Sha256 makes SHA-256 digests.
type Sha256 struct{}

// Create returns an empty digest.
func (Sha256) Create() ports.Digest { return &running{hasher: sha256.New()} }

type running struct{ hasher hash.Hash }

// Write adds p to the hash.
func (d *running) Write(p []byte) (int, error) { return d.hasher.Write(p) }

// Sum returns the hash of the bytes written so far.
func (d *running) Sum() []byte { return d.hasher.Sum(nil) }

// Save returns the state of the hash.
func (d *running) Save() ([]byte, error) {
	marshaler, ok := d.hasher.(encoding.BinaryMarshaler)
	if !ok {
		return nil, errors.New("the hash cannot be saved")
	}
	state, err := marshaler.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("save the hash: %w", err)
	}
	return state, nil
}

// Restore sets the hash to a state Save returned.
func (d *running) Restore(state []byte) error {
	unmarshaler, ok := d.hasher.(encoding.BinaryUnmarshaler)
	if !ok {
		return errors.New("the hash cannot be restored")
	}
	if err := unmarshaler.UnmarshalBinary(state); err != nil {
		return fmt.Errorf("restore the hash: %w", err)
	}
	return nil
}
