package identity_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func newRecord(t *testing.T, name string) (identity.Record, account) {
	t.Helper()
	a := newChain(t)
	device := newDevice(t)
	chain, err := appended(a.chain, a.recovery, device.Public(), name, epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := a.keys.WrapForDevice(device.Public())
	if err != nil {
		t.Fatal(err)
	}
	return identity.Record{Device: device, Wrapped: wrapped, Chain: chain}, a
}

func TestParseRecord_RoundTrip(t *testing.T) {
	t.Parallel()
	record, a := newRecord(t, "Laptop")
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
	if len(parsed.Chain.ListDevices()) != 1 {
		t.Errorf("devices of the chain = %+v", parsed.Chain.ListDevices())
	}
}

func TestParseRecord_RefusesADeviceThatIsNotInItsChain(t *testing.T) {
	t.Parallel()
	record, _ := newRecord(t, "Laptop")
	record.Device = newDevice(t)
	if _, err := identity.ParseRecord(record.Bytes()); !errors.Is(err, identity.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}

func TestParseRecord_RejectsAnotherVersionAndTruncatedBytes(t *testing.T) {
	t.Parallel()
	record, _ := newRecord(t, "Laptop")
	b := record.Bytes()
	other := bytes.Clone(b)
	other[0] = 0x02
	if _, err := identity.ParseRecord(other); !errors.Is(err, identity.ErrUnsupportedVersion) {
		t.Errorf("version 2: error = %v, want ErrUnsupportedVersion", err)
	}
	for name, cut := range map[string][]byte{"no bytes": nil, "cut": b[:50], "trailing": append(bytes.Clone(b), 0)} {
		if _, err := identity.ParseRecord(cut); !errors.Is(err, identity.ErrMalformed) {
			t.Errorf("%s: error = %v, want ErrMalformed", name, err)
		}
	}
}

func TestParseRecord_RefusesAChainWithABadSignature(t *testing.T) {
	t.Parallel()
	record, _ := newRecord(t, "Laptop")
	b := record.Bytes()
	at := bytes.Index(b, []byte("Laptop"))
	b[at] ^= 0x01
	if _, err := identity.ParseRecord(b); !errors.Is(err, identity.ErrBadSignature) {
		t.Errorf("error = %v, want ErrBadSignature", err)
	}
}

// The key store of the desktop keeps at most 2560 bytes of a value (apps/desktop/internal/keychain).
func TestRecord_FitsTheKeyStoreWithTheLongestDeviceName(t *testing.T) {
	t.Parallel()
	record, _ := newRecord(t, strings.Repeat("n", 64))
	value := base64.RawURLEncoding.EncodeToString(record.Bytes())
	if len(value) >= 2560 {
		t.Errorf("the key store value has %d characters, the limit is 2560", len(value))
	}
}

func TestRecord_ContainsNoWord(t *testing.T) {
	t.Parallel()
	words := abandonWords(t)
	record, _ := newRecord(t, "Laptop")
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
