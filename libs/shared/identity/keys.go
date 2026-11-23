package identity

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hpke"
	"fmt"
	"io"
)

const (
	wrappedSize = 32 + 64 + 16
	plainSize   = 64
)

func signPublicKey(seed [32]byte) ed25519.PublicKey {
	private := ed25519.NewKeyFromSeed(seed[:])
	return ed25519.PublicKey(append([]byte(nil), private[32:]...))
}

func signWith(seed [32]byte, message []byte) []byte {
	return ed25519.Sign(ed25519.NewKeyFromSeed(seed[:]), message)
}

func deriveBoxPublicKey(private [32]byte) ([32]byte, error) {
	key, err := ecdh.X25519().NewPrivateKey(private[:])
	if err != nil {
		return [32]byte{}, fmt.Errorf("identity: X25519 private key: %w", err)
	}
	return [32]byte(key.PublicKey().Bytes()), nil
}

// RecoveryKey is the root of the device chain, derived from the recovery words.
type RecoveryKey struct {
	signSeed [32]byte
	boxKey   [32]byte
	public   RecoveryPublicKey
}

type RecoveryPublicKey struct {
	Sign ed25519.PublicKey
	Box  [32]byte
}

func newRecoveryKey(signSeed, boxKey [32]byte) (RecoveryKey, error) {
	box, err := deriveBoxPublicKey(boxKey)
	if err != nil {
		return RecoveryKey{}, err
	}
	return RecoveryKey{signSeed: signSeed, boxKey: boxKey, public: RecoveryPublicKey{Sign: signPublicKey(signSeed), Box: box}}, nil
}

func (k RecoveryKey) Public() RecoveryPublicKey { return k.public }

func (k RecoveryKey) UnwrapAccountKeys(w WrappedAccountKeys) (AccountKeys, error) {
	return unwrapAccountKeys(k.boxKey, w)
}

func (k RecoveryKey) SignerKey() ed25519.PublicKey { return k.public.Sign }

func (k RecoveryKey) signEntry(message []byte) []byte { return signWith(k.signSeed, message) }

// DeviceKey is random per device and never leaves it.
type DeviceKey struct {
	seed   [32]byte
	public DevicePublicKey
}

type DevicePublicKey struct {
	Sign ed25519.PublicKey
	Box  [32]byte
}

func NewDeviceKey(random io.Reader) (DeviceKey, error) {
	var seed [32]byte
	if _, err := io.ReadFull(random, seed[:]); err != nil {
		return DeviceKey{}, fmt.Errorf("identity: read device seed: %w", err)
	}
	return newDeviceKey(seed)
}

func newDeviceKey(seed [32]byte) (DeviceKey, error) {
	key := DeviceKey{seed: seed}
	box, err := deriveBoxPublicKey(key.boxKey())
	if err != nil {
		return DeviceKey{}, err
	}
	key.public = DevicePublicKey{Sign: signPublicKey(key.signSeed()), Box: box}
	return key, nil
}

func (d DeviceKey) signSeed() [32]byte { return deriveKey(d.seed[:], labelDeviceSign) }

func (d DeviceKey) boxKey() [32]byte { return deriveKey(d.seed[:], labelDeviceBox) }

func (d DeviceKey) Public() DevicePublicKey { return d.public }

func (d DeviceKey) UnwrapAccountKeys(w WrappedAccountKeys) (AccountKeys, error) {
	return unwrapAccountKeys(d.boxKey(), w)
}

func (d DeviceKey) SignerKey() ed25519.PublicKey { return d.public.Sign }

func (d DeviceKey) signEntry(message []byte) []byte { return signWith(d.signSeed(), message) }

// AccountKeys are random and the same on every device of the account.
type AccountKeys struct {
	box       [32]byte
	kek       [32]byte
	boxPublic [32]byte
	epoch     uint32
	earlier   map[uint32][32]byte
}

func NewAccountKeys(random io.Reader) (AccountKeys, error) {
	var keys AccountKeys
	if _, err := io.ReadFull(random, keys.box[:]); err != nil {
		return AccountKeys{}, fmt.Errorf("identity: read account box key: %w", err)
	}
	if _, err := io.ReadFull(random, keys.kek[:]); err != nil {
		return AccountKeys{}, fmt.Errorf("identity: read account KEK: %w", err)
	}
	public, err := deriveBoxPublicKey(keys.box)
	if err != nil {
		return AccountKeys{}, err
	}
	keys.boxPublic = public
	return keys, nil
}

func (a AccountKeys) BoxPublicKey() [32]byte { return a.boxPublic }

func (a AccountKeys) WrapForRecovery(to RecoveryPublicKey) (WrappedAccountKeys, error) {
	return wrapAccountKeys(a, to.Box)
}

func (a AccountKeys) WrapForDevice(to DevicePublicKey) (WrappedAccountKeys, error) {
	return wrapAccountKeys(a, to.Box)
}

// WrappedAccountKeys is the HPKE output enc followed by the ciphertext.
type WrappedAccountKeys struct{ sealed []byte }

func ParseWrappedAccountKeys(b []byte) (WrappedAccountKeys, error) {
	if len(b) != wrappedSize {
		return WrappedAccountKeys{}, fmt.Errorf("%w: wrapped account keys have %d bytes, want %d", ErrMalformed, len(b), wrappedSize)
	}
	return WrappedAccountKeys{sealed: append([]byte(nil), b...)}, nil
}

func (w WrappedAccountKeys) Bytes() []byte { return append([]byte(nil), w.sealed...) }

func wrapAccountKeys(keys AccountKeys, recipientBox [32]byte) (WrappedAccountKeys, error) {
	public, err := ecdh.X25519().NewPublicKey(recipientBox[:])
	if err != nil {
		return WrappedAccountKeys{}, fmt.Errorf("identity: recipient key: %w", err)
	}
	recipient, err := hpke.NewDHKEMPublicKey(public)
	if err != nil {
		return WrappedAccountKeys{}, fmt.Errorf("identity: recipient key: %w", err)
	}
	plain := make([]byte, 0, plainSize)
	plain = append(plain, keys.box[:]...)
	plain = append(plain, keys.kek[:]...)
	defer clear(plain)
	sealed, err := hpke.Seal(recipient, hpke.HKDFSHA256(), hpke.ChaCha20Poly1305(), []byte(labelAccountKeys), plain)
	if err != nil {
		return WrappedAccountKeys{}, fmt.Errorf("identity: wrap account keys: %w", err)
	}
	return WrappedAccountKeys{sealed: sealed}, nil
}

func unwrapAccountKeys(boxKey [32]byte, w WrappedAccountKeys) (AccountKeys, error) {
	private, err := ecdh.X25519().NewPrivateKey(boxKey[:])
	if err != nil {
		return AccountKeys{}, fmt.Errorf("%w: %w", ErrUnwrap, err)
	}
	recipient, err := hpke.NewDHKEMPrivateKey(private)
	if err != nil {
		return AccountKeys{}, fmt.Errorf("%w: %w", ErrUnwrap, err)
	}
	plain, err := hpke.Open(recipient, hpke.HKDFSHA256(), hpke.ChaCha20Poly1305(), []byte(labelAccountKeys), w.sealed)
	if err != nil {
		return AccountKeys{}, fmt.Errorf("%w: %w", ErrUnwrap, err)
	}
	defer clear(plain)
	if len(plain) != plainSize {
		return AccountKeys{}, fmt.Errorf("%w: %d bytes of keys", ErrUnwrap, len(plain))
	}
	var keys AccountKeys
	copy(keys.box[:], plain[:32])
	copy(keys.kek[:], plain[32:])
	public, err := deriveBoxPublicKey(keys.box)
	if err != nil {
		return AccountKeys{}, fmt.Errorf("%w: %w", ErrUnwrap, err)
	}
	keys.boxPublic = public
	return keys, nil
}
