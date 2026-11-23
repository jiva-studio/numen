package identity

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"slices"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxDeviceName = 64

type EntryKind uint8

const (
	KindGenesis   EntryKind = 1
	KindAddDevice EntryKind = 2
)

type Entry struct {
	Seq       uint64
	Prev      [32]byte
	Kind      EntryKind
	Time      time.Time
	Signer    ed25519.PublicKey
	Genesis   *Genesis
	AddDevice *AddDevice
	Signature [64]byte
}

type Genesis struct {
	Recovery           RecoveryPublicKey
	AccountBox         [32]byte
	WrappedForRecovery WrappedAccountKeys
}

type AddDevice struct {
	Device DevicePublicKey
	Name   string
}

type Device struct {
	Public  DevicePublicKey
	Name    string
	AddedAt time.Time
}

// Signer signs a chain entry. RecoveryKey and DeviceKey implement it.
type Signer interface {
	SignerKey() ed25519.PublicKey
	signEntry(message []byte) []byte
}

// Chain is a signed, append-only log of devices rooted in the recovery signing key.
type Chain struct{ entries []Entry }

func NewChain(root RecoveryKey, account AccountKeys, at time.Time) (Chain, error) {
	public := root.Public()
	wrapped, err := account.WrapForRecovery(public)
	if err != nil {
		return Chain{}, err
	}
	entry := Entry{
		Kind:    KindGenesis,
		Time:    at.UTC().Truncate(time.Second),
		Signer:  public.Sign,
		Genesis: &Genesis{Recovery: public, AccountBox: account.BoxPublicKey(), WrappedForRecovery: wrapped},
	}
	sign(&entry, root)
	return Chain{entries: []Entry{entry}}, nil
}

// Len returns the number of entries in the chain.
func (c Chain) Len() int { return len(c.entries) }

// IsExtensionOf reports whether the first prefix.Len() entries of c equal the entries of prefix.
// An empty prefix is not a prefix of anything.
func (c Chain) IsExtensionOf(prefix Chain) bool {
	if len(prefix.entries) == 0 || len(prefix.entries) > len(c.entries) {
		return false
	}
	for i, entry := range prefix.entries {
		if hashEntry(entry) != hashEntry(c.entries[i]) {
			return false
		}
	}
	return true
}

// AppendDevice returns a new chain; the receiver is not changed.
func (c Chain) AppendDevice(by Signer, device DevicePublicKey, name string, at time.Time) (Chain, error) {
	if len(c.entries) == 0 {
		return Chain{}, fmt.Errorf("%w: the chain has no genesis", ErrMalformed)
	}
	if err := checkDeviceName(name); err != nil {
		return Chain{}, err
	}
	last := c.entries[len(c.entries)-1]
	entry := Entry{
		Seq:       last.Seq + 1,
		Prev:      hashEntry(last),
		Kind:      KindAddDevice,
		Time:      at.UTC().Truncate(time.Second),
		Signer:    by.SignerKey(),
		AddDevice: &AddDevice{Device: device, Name: name},
	}
	sign(&entry, by)
	next := Chain{entries: append(slices.Clone(c.entries), entry)}
	if err := next.Verify(); err != nil {
		return Chain{}, err
	}
	return next, nil
}

// Verify checks, entry by entry: the kind, the link to the entry before, the signer, the signature.
func (c Chain) Verify() error {
	if len(c.entries) == 0 {
		return fmt.Errorf("%w: the chain has no entries", ErrMalformed)
	}
	var root ed25519.PublicKey
	known := make(map[string]struct{})
	for i, entry := range c.entries {
		if (i == 0) != (entry.Kind == KindGenesis) {
			return fmt.Errorf("%w: entry %d has kind %d", ErrMalformed, i, entry.Kind)
		}
		if err := checkLink(c.entries, i); err != nil {
			return err
		}
		if i == 0 {
			var err error
			if root, err = verifyGenesis(entry); err != nil {
				return err
			}
			continue
		}
		if err := verifyAddDevice(entry, i, root, known); err != nil {
			return err
		}
	}
	return nil
}

func verifyGenesis(entry Entry) (ed25519.PublicKey, error) {
	if entry.Genesis == nil || !bytes.Equal(entry.Signer, entry.Genesis.Recovery.Sign) {
		return nil, fmt.Errorf("%w: the genesis is not signed by its recovery key", ErrMalformed)
	}
	if !hasValidSignature(entry) {
		return nil, fmt.Errorf("%w: entry 0", ErrBadSignature)
	}
	return entry.Genesis.Recovery.Sign, nil
}

func verifyAddDevice(entry Entry, i int, root ed25519.PublicKey, known map[string]struct{}) error {
	if _, ok := known[string(entry.Signer)]; !ok && !bytes.Equal(entry.Signer, root) {
		return fmt.Errorf("%w: entry %d", ErrUnknownSigner, i)
	}
	if !hasValidSignature(entry) {
		return fmt.Errorf("%w: entry %d", ErrBadSignature, i)
	}
	if entry.AddDevice == nil {
		return fmt.Errorf("%w: entry %d has no device", ErrMalformed, i)
	}
	if err := checkDeviceName(entry.AddDevice.Name); err != nil {
		return err
	}
	added := string(entry.AddDevice.Device.Sign)
	if _, ok := known[added]; ok || bytes.Equal(entry.AddDevice.Device.Sign, root) {
		return fmt.Errorf("%w: entry %d", ErrDuplicateDevice, i)
	}
	known[added] = struct{}{}
	return nil
}

func hasValidSignature(entry Entry) bool {
	return len(entry.Signer) == ed25519.PublicKeySize && ed25519.Verify(entry.Signer, encodeSignedMessage(entry), entry.Signature[:])
}

func checkLink(entries []Entry, i int) error {
	var want [32]byte
	if i > 0 {
		want = hashEntry(entries[i-1])
	}
	if entries[i].Seq != uint64(i) || entries[i].Prev != want {
		return fmt.Errorf("%w: entry %d", ErrBrokenLink, i)
	}
	return nil
}

func (c Chain) Root() RecoveryPublicKey {
	if len(c.entries) == 0 || c.entries[0].Genesis == nil {
		return RecoveryPublicKey{}
	}
	return c.entries[0].Genesis.Recovery
}

func (c Chain) AccountBoxKey() [32]byte {
	if len(c.entries) == 0 || c.entries[0].Genesis == nil {
		return [32]byte{}
	}
	return c.entries[0].Genesis.AccountBox
}

func (c Chain) WrappedForRecovery() WrappedAccountKeys {
	if len(c.entries) == 0 || c.entries[0].Genesis == nil {
		return WrappedAccountKeys{}
	}
	return c.entries[0].Genesis.WrappedForRecovery
}

func (c Chain) ListDevices() []Device {
	var devices []Device
	for _, entry := range c.entries {
		if entry.Kind == KindAddDevice && entry.AddDevice != nil {
			devices = append(devices, Device{Public: entry.AddDevice.Device, Name: entry.AddDevice.Name, AddedAt: entry.Time})
		}
	}
	return devices
}

func sign(entry *Entry, by Signer) {
	copy(entry.Signature[:], by.signEntry(encodeSignedMessage(*entry)))
}

func encodeSignedMessage(entry Entry) []byte {
	message := append([]byte(labelChainEntry), 0)
	return append(message, encodeEntryBody(entry)...)
}

func hashEntry(entry Entry) [32]byte { return sha256.Sum256(encodeEntry(entry)) }

func checkDeviceName(name string) error {
	if len(name) > maxDeviceName {
		return fmt.Errorf("%w: %d bytes, at most %d", ErrDeviceName, len(name), maxDeviceName)
	}
	if !utf8.ValidString(name) {
		return fmt.Errorf("%w: not UTF-8", ErrDeviceName)
	}
	if slices.ContainsFunc([]rune(name), unicode.IsControl) {
		return fmt.Errorf("%w: a control character", ErrDeviceName)
	}
	return nil
}
