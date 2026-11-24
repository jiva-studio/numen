package identity

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"maps"
	"slices"
	"time"
)

const (
	labelKeyHistory  = "numen/v1/key-history"
	previousKeysSize = sealNonceSize + plainSize + 16
)

// Revoke is the entry that removes a device and moves the account to the next epoch of keys.
type Revoke struct {
	Device             ed25519.PublicKey
	Epoch              uint32
	AccountBox         [32]byte
	WrappedForRecovery WrappedAccountKeys
	PreviousKeys       []byte
	Wrapped            []DeviceWrap
}

// DeviceWrap holds the keys of an epoch wrapped for one device.
type DeviceWrap struct {
	Device  ed25519.PublicKey
	Wrapped WrappedAccountKeys
}

// RevokeDevice returns a chain that revokes the device and the keys of the next epoch; the receiver is not changed.
// The current keys are sealed under the new key encryption key, so the chain keeps the history of the keys.
func (c Chain) RevokeDevice(by Signer, current AccountKeys, device ed25519.PublicKey, random io.Reader, at time.Time) (Chain, AccountKeys, error) {
	if len(c.entries) == 0 {
		return Chain{}, AccountKeys{}, fmt.Errorf("%w: the chain has no genesis", ErrMalformed)
	}
	var remaining []Device
	for _, one := range c.replay().active {
		if !bytes.Equal(one.Public.Sign, device) {
			remaining = append(remaining, one)
		}
	}
	revoke, fresh, err := c.makeKeys(current, remaining, random)
	if err != nil {
		return Chain{}, AccountKeys{}, err
	}
	revoke.Device = slices.Clone(device)
	return c.appendKeys(by, KindRevokeDevice, revoke, current, fresh, at)
}

// RotateKeys returns a chain that starts the next epoch of keys for every active device and removes none. Only the recovery key signs it:
// a restore from the words rotates, so the keys a chain wrapped for the recovery key are never the keys in use.
func (c Chain) RotateKeys(by RecoveryKey, current AccountKeys, random io.Reader, at time.Time) (Chain, AccountKeys, error) {
	if len(c.entries) == 0 {
		return Chain{}, AccountKeys{}, fmt.Errorf("%w: the chain has no genesis", ErrMalformed)
	}
	revoke, fresh, err := c.makeKeys(current, c.replay().active, random)
	if err != nil {
		return Chain{}, AccountKeys{}, err
	}
	return c.appendKeys(by, KindRotateKeys, revoke, current, fresh, at)
}

func (c Chain) appendKeys(by Signer, kind EntryKind, revoke *Revoke, current, fresh AccountKeys, at time.Time) (Chain, AccountKeys, error) {
	last := c.entries[len(c.entries)-1]
	entry := Entry{
		Seq:    last.Seq + 1,
		Prev:   hashEntry(last),
		Kind:   kind,
		Time:   at.UTC().Truncate(time.Second),
		Signer: by.SignerKey(),
		Revoke: revoke,
	}
	sign(&entry, by)
	next := Chain{entries: append(slices.Clone(c.entries), entry)}
	if err := next.Verify(); err != nil {
		return Chain{}, AccountKeys{}, err
	}
	fresh.epoch = revoke.Epoch
	fresh.earlier = maps.Clone(current.earlier)
	if fresh.earlier == nil {
		fresh.earlier = make(map[uint32][32]byte)
	}
	fresh.earlier[revoke.Epoch-1] = current.kek
	return next, fresh, nil
}

func (c Chain) makeKeys(current AccountKeys, remaining []Device, random io.Reader) (*Revoke, AccountKeys, error) {
	fresh, err := NewAccountKeys(random)
	if err != nil {
		return nil, AccountKeys{}, err
	}
	epoch := c.replay().epoch + 1
	forRecovery, err := fresh.WrapForRecovery(c.Root())
	if err != nil {
		return nil, AccountKeys{}, err
	}
	previous, err := sealPreviousKeys(random, fresh, epoch, current)
	if err != nil {
		return nil, AccountKeys{}, err
	}
	var wraps []DeviceWrap
	for _, one := range remaining {
		wrapped, err := fresh.WrapForDevice(one.Public)
		if err != nil {
			return nil, AccountKeys{}, err
		}
		wraps = append(wraps, DeviceWrap{Device: one.Public.Sign, Wrapped: wrapped})
	}
	return &Revoke{
		Epoch:              epoch,
		AccountBox:         fresh.BoxPublicKey(),
		WrappedForRecovery: forRecovery,
		PreviousKeys:       previous,
		Wrapped:            wraps,
	}, fresh, nil
}

func newHistoryAEAD(kek [32]byte) (cipher.AEAD, error) {
	key, err := hkdf.Key(sha256.New, kek[:], nil, labelKeyHistory, sealKeySize)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func getHistoryAssociatedData(epoch uint32) []byte {
	return binary.BigEndian.AppendUint32([]byte(labelKeyHistory), epoch)
}

func sealPreviousKeys(random io.Reader, fresh AccountKeys, epoch uint32, previous AccountKeys) ([]byte, error) {
	aead, err := newHistoryAEAD(fresh.kek)
	if err != nil {
		return nil, fmt.Errorf("identity: derive key history key: %w", err)
	}
	nonce := make([]byte, sealNonceSize)
	if _, err := io.ReadFull(random, nonce); err != nil {
		return nil, fmt.Errorf("identity: read key history nonce: %w", err)
	}
	plain := append(slices.Clone(previous.box[:]), previous.kek[:]...)
	defer clear(plain)
	return aead.Seal(nonce, nonce, plain, getHistoryAssociatedData(epoch)), nil
}

func openPreviousKeys(kek [32]byte, epoch uint32, sealed []byte) ([32]byte, error) {
	aead, err := newHistoryAEAD(kek)
	if err != nil || len(sealed) != previousKeysSize {
		return [32]byte{}, ErrUnwrap
	}
	plain, err := aead.Open(nil, sealed[:sealNonceSize], sealed[sealNonceSize:], getHistoryAssociatedData(epoch))
	if err != nil || len(plain) != plainSize {
		return [32]byte{}, ErrUnwrap
	}
	defer clear(plain)
	return [32]byte(plain[32:]), nil
}

// GetWrappedFor returns the keys wrapped for the device by the latest entry that wraps for it; the genesis is not looked at.
func (c Chain) GetWrappedFor(device ed25519.PublicKey) (WrappedAccountKeys, bool) {
	for i := len(c.entries) - 1; i > 0; i-- {
		revoke := c.entries[i].keyChange()
		if revoke == nil {
			continue
		}
		if wrapped, ok := revoke.getWrapFor(device); ok {
			return wrapped, true
		}
	}
	return WrappedAccountKeys{}, false
}

// OpenKeyHistory returns the keys of the latest epoch with the key encryption key of every earlier epoch.
// The keys must be the ones of the latest epoch, or the error is ErrUnwrap.
func (c Chain) OpenKeyHistory(current AccountKeys) (AccountKeys, error) {
	if current.BoxPublicKey() != c.AccountBoxKey() {
		return AccountKeys{}, fmt.Errorf("%w: the keys are not those of the latest epoch", ErrUnwrap)
	}
	return c.openHistoryFrom(current, c.Epoch())
}

func (c Chain) openHistoryFrom(current AccountKeys, epoch uint32) (AccountKeys, error) {
	keys := current
	keys.epoch = epoch
	keys.earlier = make(map[uint32][32]byte)
	kek := current.kek
	for i := len(c.entries) - 1; i > 0; i-- {
		revoke := c.entries[i].keyChange()
		if revoke == nil || revoke.Epoch > epoch {
			continue
		}
		previous, err := openPreviousKeys(kek, revoke.Epoch, revoke.PreviousKeys)
		if err != nil {
			return AccountKeys{}, err
		}
		keys.earlier[revoke.Epoch-1] = previous
		kek = previous
	}
	return keys, nil
}

func (c Chain) getKeyEpoch(sign ed25519.PublicKey) uint32 {
	var epoch, held uint32
	for _, entry := range c.entries[min(1, len(c.entries)):] {
		revoke := entry.keyChange()
		switch {
		case entry.Kind == KindAddDevice && entry.AddDevice != nil && bytes.Equal(entry.AddDevice.Device.Sign, sign):
			held = epoch
		case revoke != nil:
			if bytes.Equal(revoke.Device, sign) {
				return held
			}
			epoch = revoke.Epoch
			if _, ok := revoke.getWrapFor(sign); ok {
				held = epoch
			}
		}
	}
	return held
}

func (r *Revoke) getWrapFor(sign ed25519.PublicKey) (WrappedAccountKeys, bool) {
	for _, wrap := range r.Wrapped {
		if bytes.Equal(wrap.Device, sign) {
			return wrap.Wrapped, true
		}
	}
	return WrappedAccountKeys{}, false
}

// GetRevokedSince returns the signing keys revoked by the entries after the prefix.
func (c Chain) GetRevokedSince(prefix Chain) []ed25519.PublicKey {
	var revoked []ed25519.PublicKey
	for i := len(prefix.entries); i < len(c.entries); i++ {
		if entry := c.entries[i]; entry.Kind == KindRevokeDevice && entry.Revoke != nil {
			revoked = append(revoked, slices.Clone(entry.Revoke.Device))
		}
	}
	return revoked
}

// Epoch returns the epoch of the keys; 0 for keys that were just unwrapped, until OpenKeyHistory sets it.
func (a AccountKeys) Epoch() uint32 { return a.epoch }

func (c Chain) getKeyIndex(sign ed25519.PublicKey) int {
	found := -1
	for i := 1; i < len(c.entries); i++ {
		entry := c.entries[i]
		if entry.Kind == KindAddDevice && entry.AddDevice != nil && bytes.Equal(entry.AddDevice.Device.Sign, sign) {
			found = i
		}
		if entry.Kind == KindRevokeDevice && entry.Revoke != nil {
			if _, ok := entry.Revoke.getWrapFor(sign); ok {
				found = i
			}
		}
	}
	return found
}
