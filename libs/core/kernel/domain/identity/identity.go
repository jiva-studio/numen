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
	// ErrCannotOpen means sealed data does not open for this plugin, context and account.
	ErrCannotOpen = errors.New("the sealed data cannot be opened")
	// ErrNoPendingRestore means no restore waits for a plugin.
	ErrNoPendingRestore = errors.New("no restore is pending")
	// ErrInvalidChain means a chain does not verify or does not belong to the recovery words.
	ErrInvalidChain = errors.New("the device chain is not valid for these recovery words")
)

// Identity is what a person can see of the identity of this device.
type Identity struct {
	AccountFingerprint string
	DeviceFingerprint  string
	DeviceName         string
	DeviceAddedAt      time.Time
}

// Public is what the identity of this device shows to a plugin and to a server: fingerprints, the
// public signing key of the device and the signed device chain. It holds no secret.
type Public struct {
	AccountFingerprint string
	DeviceFingerprint  string
	DeviceSignKey      []byte
	Chain              []byte
}

// PendingRestore is what a restore that waits for the account chain shows to a plugin: public keys only.
type PendingRestore struct {
	IsPending     bool
	RootSignKey   []byte
	DeviceSignKey []byte
}
