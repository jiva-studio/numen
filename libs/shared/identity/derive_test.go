package identity_test

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha256"
	"errors"
	"regexp"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func abandonWords(t *testing.T) identity.RecoveryWords {
	t.Helper()
	return parseWords(t, bip39Vectors()[0].words)
}

type failer interface {
	Helper()
	Fatal(args ...any)
}

func deriveRecovery(t failer, words identity.RecoveryWords) identity.RecoveryKey {
	t.Helper()
	key, err := words.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func parseWords(t *testing.T, text string) identity.RecoveryWords {
	t.Helper()
	words, err := identity.ParseRecoveryWords(strings.Fields(text))
	if err != nil {
		t.Fatal(err)
	}
	return words
}

func deviceFromSeed(t *testing.T, seed []byte) identity.DeviceKey {
	t.Helper()
	device, err := identity.NewDeviceKey(bytes.NewReader(seed))
	if err != nil {
		t.Fatal(err)
	}
	return device
}

func wrapForRecovery(t *testing.T, account identity.AccountKeys, to identity.RecoveryPublicKey) identity.WrappedAccountKeys {
	t.Helper()
	wrapped, err := account.WrapForRecovery(to)
	if err != nil {
		t.Fatal(err)
	}
	return wrapped
}

func mustAppend(t *testing.T, c identity.Chain, by identity.Signer, device identity.DevicePublicKey, name string, at time.Time) identity.Chain {
	t.Helper()
	next, err := appended(c, by, device, name, at)
	if err != nil {
		t.Fatal(err)
	}
	return next
}

func expand(t *testing.T, secret []byte, info string) []byte {
	t.Helper()
	out, err := hkdf.Key(sha256.New, secret, nil, info, 32)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func x25519Public(t *testing.T, private []byte) [32]byte {
	t.Helper()
	key, err := ecdh.X25519().NewPrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	return [32]byte(key.PublicKey().Bytes())
}

func fingerprintOf(label string, sign ed25519.PublicKey) string {
	sum := sha256.Sum256(append(append([]byte(label), 0), sign...))
	const digits = "0123456789ABCDEF"
	var out strings.Builder
	for i, b := range sum[:16] {
		if i > 0 && i%2 == 0 {
			out.WriteByte(' ')
		}
		out.WriteByte(digits[b>>4])
		out.WriteByte(digits[b&15])
	}
	return out.String()
}

// The labels of the key model are frozen: a change of one must fail these tests.
func TestDeriveRecoveryKey_FollowsTheFrozenLabels(t *testing.T) {
	t.Parallel()
	entropy := make([]byte, 16)
	secret := expand(t, entropy, "numen/v1/recovery")
	public := deriveRecovery(t, abandonWords(t)).Public()

	wantSign := ed25519.PublicKey(ed25519.NewKeyFromSeed(expand(t, secret, "numen/v1/recovery/sign"))[32:])
	if !bytes.Equal(public.Sign, wantSign) {
		t.Errorf("recovery signing key = %x, want %x", public.Sign, wantSign)
	}
	if want := x25519Public(t, expand(t, secret, "numen/v1/recovery/box")); public.Box != want {
		t.Errorf("recovery box key = %x, want %x", public.Box, want)
	}
	if got, want := public.Fingerprint().String(), fingerprintOf("numen/v1/fingerprint/account", wantSign); got != want {
		t.Errorf("account fingerprint = %q, want %q", got, want)
	}
}

func TestDeriveRecoveryKey_IsDeterministicForTheSameWords(t *testing.T) {
	t.Parallel()
	first := deriveRecovery(t, abandonWords(t)).Public()
	second := deriveRecovery(t, abandonWords(t)).Public()
	if !bytes.Equal(first.Sign, second.Sign) || first.Box != second.Box {
		t.Error("the same words derive two keys")
	}
	other := parseWords(t, bip39Vectors()[1].words)
	if bytes.Equal(first.Sign, deriveRecovery(t, other).Public().Sign) {
		t.Error("different words derive the same key")
	}
}

func TestNewDeviceKey_FollowsTheFrozenLabels(t *testing.T) {
	t.Parallel()
	seed := bytes.Repeat([]byte{7}, 32)
	device, err := identity.NewDeviceKey(bytes.NewReader(seed))
	if err != nil {
		t.Fatal(err)
	}
	public := device.Public()
	wantSign := ed25519.PublicKey(ed25519.NewKeyFromSeed(expand(t, seed, "numen/v1/device/sign"))[32:])
	if !bytes.Equal(public.Sign, wantSign) {
		t.Errorf("device signing key = %x, want %x", public.Sign, wantSign)
	}
	if want := x25519Public(t, expand(t, seed, "numen/v1/device/box")); public.Box != want {
		t.Errorf("device box key = %x, want %x", public.Box, want)
	}
	if got, want := public.Fingerprint().String(), fingerprintOf("numen/v1/fingerprint/device", wantSign); got != want {
		t.Errorf("device fingerprint = %q, want %q", got, want)
	}
}

func TestNewDeviceKey_DiffersForEachRandomSeed(t *testing.T) {
	t.Parallel()
	first := deviceFromSeed(t, bytes.Repeat([]byte{1}, 32))
	second := deviceFromSeed(t, bytes.Repeat([]byte{2}, 32))
	if bytes.Equal(first.Public().Sign, second.Public().Sign) {
		t.Error("two seeds give one key")
	}
}

func TestNewKeys_FailWhenTheRandomSourceFails(t *testing.T) {
	t.Parallel()
	failing := iotest.ErrReader(errors.New("no entropy"))
	if _, err := identity.NewDeviceKey(failing); err == nil {
		t.Error("NewDeviceKey returned no error")
	}
	if _, err := identity.NewAccountKeys(failing); err == nil {
		t.Error("NewAccountKeys returned no error")
	}
}

var fingerprintFormat = regexp.MustCompile(`^([0-9A-F]{4} ){7}[0-9A-F]{4}$`)

func TestFingerprint_IsEightGroupsOfFourUpperCaseHexDigits(t *testing.T) {
	t.Parallel()
	public := deriveRecovery(t, abandonWords(t)).Public()
	if got := public.Fingerprint().String(); !fingerprintFormat.MatchString(got) {
		t.Errorf("account fingerprint %q has the wrong format", got)
	}
	var zero identity.Fingerprint
	if got := zero.String(); got != "0000 0000 0000 0000 0000 0000 0000 0000" {
		t.Errorf("zero fingerprint = %q", got)
	}
	var one identity.Fingerprint
	one[0], one[15] = 0x3F, 0xA9
	if got := one.String(); got != "3F00 0000 0000 0000 0000 0000 0000 00A9" {
		t.Errorf("fingerprint = %q", got)
	}
}

func TestFingerprint_SeparatesAnAccountFromADeviceOfTheSameKey(t *testing.T) {
	t.Parallel()
	recovery := deriveRecovery(t, abandonWords(t)).Public()
	asDevice := identity.DevicePublicKey(recovery)
	if recovery.Fingerprint() == asDevice.Fingerprint() {
		t.Error("the account and the device fingerprint of one key are equal")
	}
}
