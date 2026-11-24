// Package trust decides which devices may author what Numen Sync reads: the devices of the account chain that
// extends the chain of this device.
package trust

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

var errUntrustedChain = errors.New("the account chain of the server does not extend the chain of this device")

// Authors are the devices of the account.
type Authors struct {
	devices [][]byte
}

// NewAuthors returns the devices of the chain of the server, which must verify, have the root of the own chain
// and extend it.
func NewAuthors(own, stored []byte) (Authors, error) {
	ownChain, err := identity.ParseChain(own)
	if err != nil {
		return Authors{devices: nil}, fmt.Errorf("read the chain of this device: %w", err)
	}
	chain, err := identity.ParseChain(stored)
	if err != nil {
		return Authors{devices: nil}, fmt.Errorf("read the chain of the server: %w", err)
	}
	if !bytes.Equal(chain.Root().Sign, ownChain.Root().Sign) || !chain.IsExtensionOf(ownChain) {
		return Authors{devices: nil}, errUntrustedChain
	}
	listed := chain.ListDevices()
	devices := make([][]byte, len(listed))
	for i, device := range listed {
		devices[i] = device.Public.Sign
	}
	return Authors{devices: devices}, nil
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
