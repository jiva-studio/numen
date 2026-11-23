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

// MaxChainBytes is the longest encoded chain a device or the server accepts.
const MaxChainBytes = 192 << 10

// MaxActiveDevices is the most devices a chain lists as active.
const MaxActiveDevices = 32

// Verify checks, entry by entry: the kind, the link to the entry before, the signer, the signature, the rules of each kind and the size.
func (c Chain) Verify() error {
	if len(c.entries) == 0 {
		return fmt.Errorf("%w: the chain has no entries", ErrMalformed)
	}
	var state roster
	for i, entry := range c.entries {
		if (i == 0) != (entry.Kind == KindGenesis) {
			return fmt.Errorf("%w: entry %d has kind %d", ErrMalformed, i, entry.Kind)
		}
		if err := checkLink(c.entries, i); err != nil {
			return err
		}
		if i == 0 {
			root, err := verifyGenesis(entry)
			if err != nil {
				return err
			}
			state = newRoster(root)
			continue
		}
		if err := state.verify(entry, i); err != nil {
			return err
		}
		state.apply(entry)
	}
	if len(c.Bytes()) > MaxChainBytes {
		return fmt.Errorf("%w: more than %d bytes", ErrChainTooLong, MaxChainBytes)
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

func (e Entry) keyChange() *Revoke {
	if e.Kind == KindRevokeDevice || e.Kind == KindRotateKeys {
		return e.Revoke
	}
	return nil
}
