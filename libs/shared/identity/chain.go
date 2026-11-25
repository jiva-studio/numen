package identity

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxDeviceName = 64

type EntryKind uint8

const (
	KindGenesis      EntryKind = 1
	KindAddDevice    EntryKind = 2
	KindRevokeDevice EntryKind = 3
	KindRotateKeys   EntryKind = 4
)

type Entry struct {
	Seq       uint64
	Prev      [32]byte
	Kind      EntryKind
	Time      time.Time
	Signer    ed25519.PublicKey
	Genesis   *Genesis
	AddDevice *AddDevice
	Revoke    *Revoke
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

func getEntryTime(at time.Time, last Entry) time.Time {
	at = at.UTC().Truncate(time.Second)
	if at.Before(last.Time) {
		return last.Time
	}
	return at
}

func NewChain(root RecoveryKey, account AccountKeys, at time.Time) (Chain, error) {
	public := root.Public()
	wrapped, err := account.WrapForRecovery(public)
	if err != nil {
		return Chain{}, err
	}
	entry := Entry{
		Kind:    KindGenesis,
		Time:    at.UTC().Truncate(time.Second),
		Signer:  slices.Clone(public.Sign),
		Genesis: &Genesis{Recovery: RecoveryPublicKey{Sign: slices.Clone(public.Sign), Box: public.Box}, AccountBox: account.BoxPublicKey(), WrappedForRecovery: wrapped},
	}
	sign(&entry, root)
	return Chain{entries: []Entry{entry}}, nil
}

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
		Time:      getEntryTime(at, last),
		Signer:    slices.Clone(by.SignerKey()),
		AddDevice: &AddDevice{Device: DevicePublicKey{Sign: slices.Clone(device.Sign), Box: device.Box}, Name: name},
	}
	sign(&entry, by)
	next := Chain{entries: append(slices.Clone(c.entries), entry)}
	if err := next.Verify(); err != nil {
		return Chain{}, err
	}
	return next, nil
}

// MaxChainBytes is the longest encoded chain a device or the server accepts.
const MaxChainBytes = 192 << 10

// RecoveryReserveBytes is the last part of MaxChainBytes that only entries signed by the recovery key may fill, so a device that fills the
// chain with its own entries cannot keep the recovery key from revoking it.
const RecoveryReserveBytes = 32 << 10

// MaxActiveDevices is the most devices a chain lists as active.
const MaxActiveDevices = 32

// Verify checks, entry by entry: the kind, the link to the entry before, the signer, the signature, the rules of each kind and the size.
func (c Chain) Verify() error {
	if len(c.entries) == 0 {
		return fmt.Errorf("%w: the chain has no entries", ErrMalformed)
	}
	var state roster
	for i, entry := range c.entries {
		if err := verifyEntry(c.entries, i, &state); err != nil {
			return err
		}
		if i > 0 {
			state.apply(entry)
		}
	}
	return c.checkSize()
}

// checkSize refuses a chain of more than MaxChainBytes, and an entry that a device signed after the chain passed MaxChainBytes - RecoveryReserveBytes.
func (c Chain) checkSize() error {
	deviceLimit := MaxChainBytes - RecoveryReserveBytes
	root := c.Root().Sign
	bodies := 0
	for i, entry := range c.entries {
		size := len(encodeEntry(entry))
		bodies += size + len(binary.AppendUvarint(nil, uint64(size)))
		if total := bodies + len(binary.AppendUvarint(nil, uint64(i+1))); i > 0 && !bytes.Equal(entry.Signer, root) && total > deviceLimit {
			return fmt.Errorf("%w: entry %d of a device ends at %d bytes, at most %d", ErrChainTooLong, i, total, deviceLimit)
		}
	}
	if total := bodies + len(binary.AppendUvarint(nil, uint64(len(c.entries)))); total > MaxChainBytes {
		return fmt.Errorf("%w: more than %d bytes", ErrChainTooLong, MaxChainBytes)
	}
	return nil
}

// MaxClockSkew is how far after the clock of the adopter an entry may be dated.
const MaxClockSkew = 24 * time.Hour

// VerifyAt runs Verify and refuses an entry dated more than MaxClockSkew after now (ErrEntryTime).
func (c Chain) VerifyAt(now time.Time) error {
	if err := c.Verify(); err != nil {
		return err
	}
	for i, entry := range c.entries {
		if entry.Time.After(now.Add(MaxClockSkew)) {
			return fmt.Errorf("%w: entry %d is dated after the clock", ErrEntryTime, i)
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
	if err := checkBoxPublicKey(entry.Genesis.Recovery.Box); err != nil {
		return nil, fmt.Errorf("entry 0: %w", err)
	}
	return entry.Genesis.Recovery.Sign, nil
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
	recovery := c.entries[0].Genesis.Recovery
	return RecoveryPublicKey{Sign: slices.Clone(recovery.Sign), Box: recovery.Box}
}

// AccountBoxKey returns the account box public key of the latest epoch.
func (c Chain) AccountBoxKey() [32]byte {
	if revoke := c.latestRevoke(); revoke != nil {
		return revoke.AccountBox
	}
	if len(c.entries) == 0 || c.entries[0].Genesis == nil {
		return [32]byte{}
	}
	return c.entries[0].Genesis.AccountBox
}

func (c Chain) hasAccountBoxKey(box [32]byte) bool {
	if len(c.entries) == 0 || c.entries[0].Genesis == nil {
		return false
	}
	if c.entries[0].Genesis.AccountBox == box {
		return true
	}
	for _, entry := range c.entries[1:] {
		if revoke := entry.keyChange(); revoke != nil && revoke.AccountBox == box {
			return true
		}
	}
	return false
}

// WrappedForRecovery returns the account keys of the latest epoch wrapped for the recovery key.
func (c Chain) WrappedForRecovery() WrappedAccountKeys {
	if revoke := c.latestRevoke(); revoke != nil {
		return revoke.WrappedForRecovery
	}
	if len(c.entries) == 0 || c.entries[0].Genesis == nil {
		return WrappedAccountKeys{}
	}
	return c.entries[0].Genesis.WrappedForRecovery
}

func (c Chain) latestRevoke() *Revoke {
	for i := len(c.entries) - 1; i > 0; i-- {
		if revoke := c.entries[i].keyChange(); revoke != nil {
			return revoke
		}
	}
	return nil
}

// ListDevices returns the active devices in the order they were added.
func (c Chain) ListDevices() []Device { return c.replay().active }

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

func (c Chain) getForkIndex(other Chain) int {
	for i := range min(len(c.entries), len(other.entries)) {
		if hashEntry(c.entries[i]) != hashEntry(other.entries[i]) {
			return i
		}
	}
	return -1
}

// IsPreferredOver reports whether c replaces other at a fork: both chains have the same root, and at the first entry where they differ the
// entry of c is signed by the root while the entry of other is not. When both are signed by the root the entry with the lower hash
// wins, so every device settles on the same chain. Every entry of other from that index on is void.
func (c Chain) IsPreferredOver(other Chain) bool {
	i := c.getForkIndex(other)
	if i < 1 || !bytes.Equal(c.Root().Sign, other.Root().Sign) {
		return false
	}
	root := c.Root().Sign
	mine, theirs := c.entries[i], other.entries[i]
	isMineRoot, isTheirsRoot := bytes.Equal(mine.Signer, root), bytes.Equal(theirs.Signer, root)
	if isMineRoot && isTheirsRoot {
		mineHash, theirsHash := hashEntry(mine), hashEntry(theirs)
		return bytes.Compare(mineHash[:], theirsHash[:]) < 0
	}
	return isMineRoot
}

// CoolingOff is how long a revoke signed by a device, not by the recovery key, is shown as a notice to the other devices.
const CoolingOff = 72 * time.Hour

// DeviceRevocation is a revoke of a device that another device signed.
type DeviceRevocation struct {
	By     ed25519.PublicKey
	Device ed25519.PublicKey
	At     time.Time
}

// ListDeviceRevocations returns the revokes signed by a device, not by the recovery key, in the order of the chain.
func (c Chain) ListDeviceRevocations() []DeviceRevocation {
	var found []DeviceRevocation
	for _, entry := range c.entries[min(1, len(c.entries)):] {
		if entry.Kind != KindRevokeDevice || entry.Revoke == nil || bytes.Equal(entry.Signer, c.Root().Sign) {
			continue
		}
		found = append(found, DeviceRevocation{By: bytes.Clone(entry.Signer), Device: bytes.Clone(entry.Revoke.Device), At: entry.Time})
	}
	return found
}

// DeviceAddition is the addition of a device that another device signed.
type DeviceAddition struct {
	By     ed25519.PublicKey
	Device ed25519.PublicKey
	At     time.Time
}

// ListDeviceAdditions returns the additions signed by a device, not by the recovery key, in the order of the chain.
func (c Chain) ListDeviceAdditions() []DeviceAddition {
	var found []DeviceAddition
	for _, entry := range c.entries[min(1, len(c.entries)):] {
		if entry.Kind != KindAddDevice || entry.AddDevice == nil || bytes.Equal(entry.Signer, c.Root().Sign) {
			continue
		}
		found = append(found, DeviceAddition{By: bytes.Clone(entry.Signer), Device: bytes.Clone(entry.AddDevice.Device.Sign), At: entry.Time})
	}
	return found
}

func (e Entry) keyChange() *Revoke {
	if e.Kind == KindRevokeDevice || e.Kind == KindRotateKeys {
		return e.Revoke
	}
	return nil
}

func verifyEntry(entries []Entry, i int, state *roster) error {
	entry := entries[i]
	if (i == 0) != (entry.Kind == KindGenesis) {
		return fmt.Errorf("%w: entry %d has kind %d", ErrMalformed, i, entry.Kind)
	}
	if err := checkLink(entries, i); err != nil {
		return err
	}
	if i == 0 {
		root, err := verifyGenesis(entry)
		if err != nil {
			return err
		}
		*state = newRoster(root)
		return nil
	}
	if entry.Time.Before(entries[i-1].Time) {
		return fmt.Errorf("%w: entry %d is older than the entry before", ErrEntryTime, i)
	}
	return state.verify(entry, i)
}
