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
	// ErrInvalidSheets means the sheets entered are not sheets of one valid kit.
	ErrInvalidSheets = errors.New("the sheets are not valid")
	// ErrNotEnoughSheets means fewer than two sheets were entered.
	ErrNotEnoughSheets = errors.New("two sheets are needed")
	// ErrOtherAccount means the proof belongs to another account than the one of this device.
	ErrOtherAccount = errors.New("the proof belongs to another account")
	// ErrInvalidChainMark means the chain mark of a recovery kit is not a number, a dash and 12 hex digits.
	ErrInvalidChainMark = errors.New("the chain mark of the recovery kit is not valid")
	// ErrChainOutdated means the chain the server holds is older than the chain mark of the recovery kit, or is another history, or the
	// server holds none: the restore was refused and keeps nothing.
	ErrChainOutdated = errors.New("the server holds an older chain than the recovery kit names")
)

// Identity is what a person can see of the identity of this device.
type Identity struct {
	AccountFingerprint string
	DeviceFingerprint  string
	DeviceName         string
	DeviceAddedAt      time.Time
	IsRevoked          bool
	KeyEpoch           uint32
}

// Public is what the identity of this device shows to a plugin and to a server: fingerprints, the
// public signing key of the device and the signed device chain. It holds no secret.
type Public struct {
	AccountFingerprint string
	DeviceFingerprint  string
	DeviceSignKey      []byte
	Chain              []byte
	KeyEpoch           uint32
	ChainLength        int
	IsRevoked          bool
}

// PendingRestore is what a restore that waits for the account chain shows to a plugin: public keys only.
type PendingRestore struct {
	IsPending     bool
	RootSignKey   []byte
	DeviceSignKey []byte
}

// Kit is three sheets of one recovery kit for the account of this device, made at CreatedAt.
type Kit struct {
	ID                 string
	AccountFingerprint string
	ChainMark          string
	CreatedAt          time.Time
	Sheets             []Sheet
}

// Sheet is one sheet of a kit: its number and its 33 words.
type Sheet struct {
	Number int
	Words  []string
}
