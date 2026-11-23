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
	// ErrInvalidSheets means the sheets entered are not sheets of one valid kit.
	ErrInvalidSheets = errors.New("the sheets are not valid")
	// ErrNotEnoughSheets means fewer than two sheets were entered.
	ErrNotEnoughSheets = errors.New("two sheets are needed")
	// ErrOtherAccount means the proof belongs to another account than the one of this device.
	ErrOtherAccount = errors.New("the proof belongs to another account")
)

// Identity is what a person can see of the identity of this device.
type Identity struct {
	AccountFingerprint string
	DeviceFingerprint  string
	DeviceName         string
	DeviceAddedAt      time.Time
}

// Kit is three sheets of one recovery kit for the account of this device, made at CreatedAt.
type Kit struct {
	ID                 string
	AccountFingerprint string
	CreatedAt          time.Time
	Sheets             []Sheet
}

// Sheet is one sheet of a kit: its number and its 33 words.
type Sheet struct {
	Number int
	Words  []string
}
