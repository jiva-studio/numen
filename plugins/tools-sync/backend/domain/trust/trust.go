// Package trust decides which devices may author what Numen Sync reads: the devices of the account chain that
// extends the chain of this device.
package trust

import (
	"bytes"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

var errUntrustedChain = errors.New("the account chain of the server does not extend the chain of this device")

// Authors are the devices of the account.
type Authors struct {
	devices [][]byte
	known   [][]byte
}

// NewAuthors returns the devices of the chain of the server, which must verify, have the root of the own chain
// and extend it or outrank it at a fork, where a recovery-signed entry replaces device entries. A chain that is a strict
// prefix of the own one is a server that went back: the devices of the own chain are returned.
func NewAuthors(own, stored []byte) (Authors, error) {
	ownChain, err := identity.ParseChain(own)
	if err != nil {
		return Authors{devices: nil, known: nil}, fmt.Errorf("read the chain of this device: %w", err)
	}
	chain, err := identity.ParseChain(stored)
	if err != nil {
		return Authors{devices: nil, known: nil}, fmt.Errorf("read the chain of the server: %w", err)
	}
	if !bytes.Equal(chain.Root().Sign, ownChain.Root().Sign) {
		return Authors{devices: nil, known: nil}, errUntrustedChain
	}
	switch {
	case chain.IsExtensionOf(ownChain), chain.Outranks(ownChain):
	case ownChain.IsExtensionOf(chain):
		chain = ownChain
	default:
		return Authors{devices: nil, known: nil}, errUntrustedChain
	}
	listed := chain.ListDevices()
	devices := make([][]byte, len(listed))
	for i, device := range listed {
		devices[i] = device.Public.Sign
	}
	return Authors{devices: devices}, nil
}

// IsKnown reports whether the key is the signing key of a device the chain ever listed, active or revoked. A revoked device is never
// trusted as an author; the key is known only so that what it wrote before the revoke, which a trusted device vouches for, can be read.
func (a Authors) IsKnown(signKey []byte) bool {
	return slices.ContainsFunc(a.known, func(device []byte) bool { return bytes.Equal(device, signKey) })
}

// IsTrusted reports whether the key is the signing key of a device of the account.
func (a Authors) IsTrusted(signKey []byte) bool {
	for _, device := range a.devices {
		if bytes.Equal(device, signKey) {
			return true
		}
	}
	return false
}
