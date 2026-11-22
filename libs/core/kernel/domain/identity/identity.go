// Package identity is the account identity of this device as the person sees it: two fingerprints
// and the name of the device. The keys behind it stay in the core.
package identity

import (
	"errors"
	"time"
)

var (
	// ErrNoIdentity means this device holds no identity.
	ErrNoIdentity = errors.New("this device has no identity")
	// ErrIdentityExists means this device already holds an identity, which creating never overwrites.
	ErrIdentityExists = errors.New("this device already has an identity")
	// ErrInvalidWords means the recovery words are not a valid set of 12 words.
	ErrInvalidWords = errors.New("the recovery words are not valid")
	// ErrKeyStore means the key store of the machine refused to read or keep the identity.
	ErrKeyStore = errors.New("the key store failed")
)

// Identity is what a person can see of the identity of this device.
type Identity struct {
	AccountFingerprint string
	DeviceFingerprint  string
	DeviceName         string
	DeviceAddedAt      time.Time
}
