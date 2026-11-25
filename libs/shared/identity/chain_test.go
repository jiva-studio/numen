package identity_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func epoch() time.Time { return time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC) }

var (
	_ identity.Signer = identity.RecoveryKey{}
	_ identity.Signer = identity.DeviceKey{}
)

type account struct {
	recovery identity.RecoveryKey
	keys     identity.AccountKeys
	chain    identity.Chain
}

func newChain(t *testing.T) account {
	t.Helper()
	recovery := deriveRecovery(t, abandonWords(t))
	keys := newAccount(t)
	chain, err := identity.NewChain(recovery, keys, epoch())
	if err != nil {
		t.Fatal(err)
	}
	return account{recovery: recovery, keys: keys, chain: chain}
}

func appended(c identity.Chain, by identity.Signer, device identity.DevicePublicKey, name string, at time.Time) (identity.Chain, error) {
	next, err := c.AppendDevice(by, device, name, at)
	if err != nil {
		return identity.Chain{}, err
	}
	return next, next.Verify()
}

func split(t *testing.T, b []byte) [][]byte {
	t.Helper()
	count, n := binary.Uvarint(b)
	b = b[n:]
	entries := make([][]byte, 0, count)
	for range count {
		size, n := binary.Uvarint(b)
		entries = append(entries, b[n:n+int(size)])
		b = b[n+int(size):]
	}
	if len(b) != 0 {
		t.Fatalf("%d bytes after the entries", len(b))
	}
	return entries
}

func join(entries [][]byte) []byte {
	out := binary.AppendUvarint(nil, uint64(len(entries)))
	for _, e := range entries {
		out = binary.AppendUvarint(out, uint64(len(e)))
		out = append(out, e...)
	}
	return out
}

func TestNewChain_StartsWithTheAccountGenesis(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	if err := a.chain.Verify(); err != nil {
		t.Fatalf("Verify() = %v", err)
	}
	root := a.chain.Root()
	want := a.recovery.Public()
	if !bytes.Equal(root.Sign, want.Sign) || root.Box != want.Box {
		t.Error("Root() is not the recovery public key")
	}
	if a.chain.AccountBoxKey() != a.keys.BoxPublicKey() {
		t.Error("AccountBoxKey() is not the account box key")
	}
	opened, err := a.recovery.UnwrapAccountKeys(a.chain.WrappedForRecovery())
	if err != nil || opened.BoxPublicKey() != a.keys.BoxPublicKey() {
		t.Errorf("the genesis keys do not open with the recovery key: %v", err)
	}
	if len(a.chain.ListDevices()) != 0 {
		t.Errorf("a new chain lists devices: %v", a.chain.ListDevices())
	}
}

func TestAppendDevice_ListsTheDeviceWithItsNameAndTime(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	device := newDevice(t)
	at := epoch().Add(1500 * time.Millisecond)
	chain, err := appended(a.chain, a.recovery, device.Public(), "Laptop", at)
	if err != nil {
		t.Fatal(err)
	}
	devices := chain.ListDevices()
	if len(devices) != 1 || devices[0].Name != "Laptop" || !bytes.Equal(devices[0].Public.Sign, device.Public().Sign) {
		t.Fatalf("ListDevices() = %+v", devices)
	}
	if want := epoch().Add(time.Second); !devices[0].AddedAt.Equal(want) {
		t.Errorf("AddedAt = %v, want %v (whole seconds)", devices[0].AddedAt, want)
	}
	if len(a.chain.ListDevices()) != 0 {
		t.Error("AppendDevice changed the chain it was called on")
	}
}

func TestAppendDevice_AcceptsADeviceAddedBeforeAsSigner(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	first, second := newDevice(t), newDevice(t)
	chain, err := appended(a.chain, a.recovery, first.Public(), "First", epoch())
	if err != nil {
		t.Fatal(err)
	}
	chain, err = appended(chain, first, second.Public(), "Second", epoch().Add(time.Minute))
	if err != nil {
		t.Fatalf("an entry signed by an added device: %v", err)
	}
	if len(chain.ListDevices()) != 2 {
		t.Errorf("ListDevices() = %+v", chain.ListDevices())
	}
}

func TestAppendDevice_RefusesTheSameDeviceTwice(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	device := newDevice(t)
	chain, err := appended(a.chain, a.recovery, device.Public(), "Laptop", epoch())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appended(chain, a.recovery, device.Public(), "Laptop again", epoch()); !errors.Is(err, identity.ErrDuplicateDevice) {
		t.Errorf("error = %v, want ErrDuplicateDevice", err)
	}
}

func TestAppendDevice_RefusesASignerThatIsNotInTheChain(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	stranger := newDevice(t)
	if _, err := appended(a.chain, stranger, newDevice(t).Public(), "Phone", epoch()); !errors.Is(err, identity.ErrUnknownSigner) {
		t.Errorf("a device that was never added signs: error = %v, want ErrUnknownSigner", err)
	}
	other := parseWords(t, bip39Vectors()[1].words)
	if _, err := appended(a.chain, deriveRecovery(t, other), newDevice(t).Public(), "Phone", epoch()); !errors.Is(err, identity.ErrUnknownSigner) {
		t.Errorf("another recovery key signs: error = %v, want ErrUnknownSigner", err)
	}
}

func TestAppendDevice_RefusesABadName(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	for name, text := range map[string]string{
		"65 bytes":        strings.Repeat("n", 65),
		"a control char":  "lap\ntop",
		"invalid UTF-8":   "lap\xfftop",
		"a tab character": "lap\ttop",
	} {
		if _, err := appended(a.chain, a.recovery, newDevice(t).Public(), text, epoch()); !errors.Is(err, identity.ErrDeviceName) {
			t.Errorf("%s: error = %v, want ErrDeviceName", name, err)
		}
	}
	if _, err := appended(a.chain, a.recovery, newDevice(t).Public(), strings.Repeat("н", 32), epoch()); err != nil {
		t.Errorf("a name of 64 bytes is refused: %v", err)
	}
}

func TestParseChain_RoundTrip(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	chain, err := appended(a.chain, a.recovery, newDevice(t).Public(), "Laptop", epoch())
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := identity.ParseChain(chain.Bytes())
	if err != nil {
		t.Fatalf("ParseChain() = %v", err)
	}
	if !bytes.Equal(parsed.Bytes(), chain.Bytes()) {
		t.Error("the parsed chain writes other bytes")
	}
	if len(parsed.ListDevices()) != 1 || parsed.ListDevices()[0].Name != "Laptop" || parsed.AccountBoxKey() != a.keys.BoxPublicKey() {
		t.Errorf("parsed chain = %+v", parsed.ListDevices())
	}
}

func TestParseChain_RefusesAnyChangedByte(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	chain := mustAppend(t, a.chain, a.recovery, newDevice(t).Public(), "Laptop", epoch())
	b := chain.Bytes()
	for i := range b {
		changed := bytes.Clone(b)
		changed[i] ^= 0x01
		if _, err := identity.ParseChain(changed); err == nil {
			t.Fatalf("byte %d of %d changed and the chain was accepted", i, len(b))
		}
	}
}

func TestParseChain_NamesABadSignatureWhereTheSignedFieldsChange(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	device := newDevice(t)
	chain := mustAppend(t, a.chain, a.recovery, device.Public(), "Laptop", epoch())
	b := chain.Bytes()
	seconds := binary.BigEndian.AppendUint64(nil, uint64(epoch().Unix()))
	places := map[string]int{
		"the device name":     bytes.Index(b, []byte("Laptop")),
		"the time":            bytes.LastIndex(b, seconds),
		"the account box key": bytes.Index(b, func() []byte { k := a.keys.BoxPublicKey(); return k[:] }()),
		"the last signature":  len(b) - 1,
	}
	for what, at := range places {
		if at < 0 {
			t.Fatalf("%s is not in the chain bytes", what)
		}
		changed := bytes.Clone(b)
		changed[at] ^= 0x01
		if _, err := identity.ParseChain(changed); !errors.Is(err, identity.ErrBadSignature) {
			t.Errorf("%s changed: error = %v, want ErrBadSignature", what, err)
		}
	}
}

func threeEntries(t *testing.T) ([][]byte, account) {
	t.Helper()
	a := newChain(t)
	chain, err := appended(a.chain, a.recovery, newDevice(t).Public(), "One", epoch())
	if err != nil {
		t.Fatal(err)
	}
	chain, err = appended(chain, a.recovery, newDevice(t).Public(), "Two", epoch())
	if err != nil {
		t.Fatal(err)
	}
	return split(t, chain.Bytes()), a
}

func TestParseChain_RefusesARemovedOrReorderedEntry(t *testing.T) {
	t.Parallel()
	entries, _ := threeEntries(t)
	removed := join([][]byte{entries[0], entries[2]})
	reordered := join([][]byte{entries[0], entries[2], entries[1]})
	for name, b := range map[string][]byte{"a removed entry": removed, "reordered entries": reordered} {
		if _, err := identity.ParseChain(b); !errors.Is(err, identity.ErrBrokenLink) {
			t.Errorf("%s: error = %v, want ErrBrokenLink", name, err)
		}
	}
}

func TestParseChain_RefusesASecondGenesis(t *testing.T) {
	t.Parallel()
	entries, _ := threeEntries(t)
	other := newChain(t)
	foreign := split(t, other.chain.Bytes())[0]
	if _, err := identity.ParseChain(join([][]byte{entries[0], foreign})); !errors.Is(err, identity.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}

func TestParseChain_RefusesBytesAfterTheEntries(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	if _, err := identity.ParseChain(append(a.chain.Bytes(), 0)); !errors.Is(err, identity.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}

func TestParseChain_RefusesAnotherVersion(t *testing.T) {
	t.Parallel()
	entries, _ := threeEntries(t)
	entries[0][0] = 0x02
	if _, err := identity.ParseChain(join(entries)); !errors.Is(err, identity.ErrUnsupportedVersion) {
		t.Errorf("error = %v, want ErrUnsupportedVersion", err)
	}
}

func TestParseChain_RejectsNoEntriesAndTruncatedBytes(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	for name, b := range map[string][]byte{"no bytes": nil, "no entries": {0}, "cut chain": a.chain.Bytes()[:40]} {
		if _, err := identity.ParseChain(b); !errors.Is(err, identity.ErrMalformed) {
			t.Errorf("%s: error = %v, want ErrMalformed", name, err)
		}
	}
}

func TestLen_CountsEntries(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	longer, err := appended(a.chain, a.recovery, newDevice(t).Public(), "Laptop", epoch().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if got := (identity.Chain{}).Len(); got != 0 {
		t.Errorf("empty Len() = %d, want 0", got)
	}
	if got := a.chain.Len(); got != 1 {
		t.Errorf("genesis Len() = %d, want 1", got)
	}
	if got := longer.Len(); got != 2 {
		t.Errorf("Len() = %d, want 2", got)
	}
}

func TestIsExtensionOf(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	laptop, err := appended(a.chain, a.recovery, newDevice(t).Public(), "Laptop", epoch().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	phone, err := appended(a.chain, a.recovery, newDevice(t).Public(), "Phone", epoch().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	other := newChain(t)
	tests := []struct {
		name   string
		chain  identity.Chain
		prefix identity.Chain
		want   bool
	}{
		{"longer chain with the shorter as prefix", laptop, a.chain, true},
		{"equal chains", laptop, laptop, true},
		{"prefix longer than the chain", a.chain, laptop, false},
		{"fork with a different entry", laptop, phone, false},
		{"chain of another account", laptop, other.chain, false},
		{"empty prefix", laptop, identity.Chain{}, false},
		{"empty chain and empty prefix", identity.Chain{}, identity.Chain{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.chain.IsExtensionOf(tt.prefix); got != tt.want {
				t.Errorf("IsExtensionOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppendDevice_RefusesADeviceWithoutASigningKey(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	if _, err := a.chain.AppendDevice(a.recovery, identity.DevicePublicKey{}, "Laptop", epoch().Add(time.Hour)); !errors.Is(err, identity.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}

func TestChain_KeepsItsKeysWhenACallerChangesTheSlicesItGaveOrTook(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	device := newDevice(t)
	public := device.Public()
	chain, err := appended(a.chain, a.recovery, public, "Laptop", epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	public.Sign[0] ^= 0xff
	chain.Root().Sign[0] ^= 0xff
	chain.ListDevices()[0].Public.Sign[0] ^= 0xff
	if err := chain.Verify(); err != nil {
		t.Fatalf("Verify() after the callers changed their slices = %v", err)
	}
}

func TestAppendDevice_RefusesABoxKeyThatCannotTakePartInAKeyAgreement(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	lowOrder := [32]byte{1}
	for name, box := range map[string][32]byte{"all zero": {}, "the point of order one": lowOrder} {
		device := identity.DevicePublicKey{Sign: newDevice(t).Public().Sign, Box: box}
		if _, err := appended(a.chain, a.recovery, device, "Phone", epoch()); !errors.Is(err, identity.ErrMalformed) {
			t.Errorf("%s: error = %v, want ErrMalformed", name, err)
		}
	}
}
