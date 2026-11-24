package identity_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func newRecord(tb testing.TB) (identity.Record, account) {
	tb.Helper()
	a := newChainOf(tb)
	device := newDeviceOf(tb)
	wrapped, err := a.keys.WrapForDevice(device.Public())
	if err != nil {
		tb.Fatal(err)
	}
	return identity.Record{Device: device, Wrapped: wrapped}, a
}

func TestParseRecord_RoundTrip(t *testing.T) {
	t.Parallel()
	record, a := newRecord(t)
	parsed, err := identity.ParseRecord(record.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Bytes(), record.Bytes()) {
		t.Error("the parsed record writes other bytes")
	}
	if !bytes.Equal(parsed.Device.Public().Sign, record.Device.Public().Sign) {
		t.Error("the device key changed")
	}
	opened, err := parsed.Device.UnwrapAccountKeys(parsed.Wrapped)
	if err != nil || opened.BoxPublicKey() != a.keys.BoxPublicKey() {
		t.Errorf("the device cannot open the keys of its record: %v", err)
	}
}

func TestRecord_ContainsOnlySeedAndWrappedKeys(t *testing.T) {
	t.Parallel()
	record, _ := newRecord(t)
	b := record.Bytes()
	if b[0] != 0x02 {
		t.Errorf("version byte = %#x, want 0x02", b[0])
	}
	if want := 1 + 32 + 1 + 112; len(b) != want {
		t.Errorf("the record has %d bytes, want %d: no chain in it", len(b), want)
	}
}

func TestParseRecord_RejectsAnotherVersionAndTruncatedBytes(t *testing.T) {
	t.Parallel()
	record, _ := newRecord(t)
	b := record.Bytes()
	for _, version := range []byte{0x00, 0x01, 0x03} {
		other := bytes.Clone(b)
		other[0] = version
		if _, err := identity.ParseRecord(other); !errors.Is(err, identity.ErrUnsupportedVersion) {
			t.Errorf("version %d: error = %v, want ErrUnsupportedVersion", version, err)
		}
	}
	for name, cut := range map[string][]byte{"no bytes": nil, "cut": b[:50], "trailing": append(bytes.Clone(b), 0)} {
		if _, err := identity.ParseRecord(cut); !errors.Is(err, identity.ErrMalformed) {
			t.Errorf("%s: error = %v, want ErrMalformed", name, err)
		}
	}
}

// The key store of the desktop keeps at most 2560 bytes of a value (apps/desktop/internal/keychain).
func TestRecord_FitsTheKeyStoreWithRoomToSpare(t *testing.T) {
	t.Parallel()
	record, _ := newRecord(t)
	value := base64.RawURLEncoding.EncodeToString(record.Bytes())
	if len(value) > 256 {
		t.Errorf("the key store value has %d characters, want at most 256", len(value))
	}
}

func TestRecord_ContainsNoWord(t *testing.T) {
	t.Parallel()
	words := abandonWords(t)
	record, _ := newRecord(t)
	b := record.Bytes()
	for _, word := range words.List() {
		if bytes.Contains(b, []byte(word)) {
			t.Errorf("the record holds the word %q", word)
		}
	}
}

func TestParseRecord_RefusesWrappedKeysThatTheDeviceCannotOpen(t *testing.T) {
	t.Parallel()
	record, a := newRecord(t, "Laptop")
	wrapped, err := a.keys.WrapForDevice(newDevice(t).Public())
	if err != nil {
		t.Fatal(err)
	}
	record.Wrapped = wrapped
	if _, err := identity.ParseRecord(record.Bytes()); !errors.Is(err, identity.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}

func TestParseRecord_RefusesWrappedKeysOfAnotherAccount(t *testing.T) {
	t.Parallel()
	record, _ := newRecord(t, "Laptop")
	wrapped, err := newAccount(t).WrapForDevice(record.Device.Public())
	if err != nil {
		t.Fatal(err)
	}
	record.Wrapped = wrapped
	if _, err := identity.ParseRecord(record.Bytes()); !errors.Is(err, identity.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}
