package identity_test

import (
	"bytes"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func getNow() time.Time { return time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC) }

const keyName = "identity"

func getLegalWords() []string {
	return strings.Fields("legal winner thank year wave sausage worth useful legal winner thank yellow")
}

var fingerprint = regexp.MustCompile(`^([0-9A-F]{4} ){7}[0-9A-F]{4}$`)

type kit struct {
	keys     *fakeKeyStore
	get      *identity.GetIdentity
	generate *identity.GenerateRecoveryWords
	create   *identity.CreateIdentity
	delete   *identity.DeleteIdentity

	saveKit *identity.SaveRecoveryKit
	saver   *fakeKitSaver
}

func newKit(name string) kit {
	keys := newFakeKeyStore()
	saver := &fakeKitSaver{}
	deps := identity.NewDeps(keys, fakeNames{name: name}, testsupport.NewClockAt(getNow()))
	return kit{
		keys:     keys,
		get:      identity.NewGetIdentity(deps),
		generate: identity.NewGenerateRecoveryWords(deps),
		create:   identity.NewCreateIdentity(deps),
		delete:   identity.NewDeleteIdentity(deps),
		saveKit:  identity.NewSaveRecoveryKit(deps, saver),
		saver:    saver,
	}
}

func TestGetIdentity_NoRecord(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	if _, err := k.get.Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Fatalf("error = %v, want ErrNoIdentity", err)
	}
}

func TestGetIdentity_ReportsARecordThatDoesNotParseAndNeverCallsItAbsent(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	k.keys.keys[keyName] = "not-a-record"
	_, err := k.get.Execute(t.Context())
	if !errors.Is(err, identitylib.ErrMalformed) || errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Fatalf("error = %v, want ErrMalformed and not ErrNoIdentity", err)
	}
	if _, err := k.create.Execute(t.Context(), getLegalWords()); !errors.Is(err, identitydomain.ErrIdentityExists) {
		t.Errorf("create over a broken record: error = %v, want ErrIdentityExists", err)
	}
	if k.keys.keys[keyName] != "not-a-record" {
		t.Error("the broken record was overwritten")
	}
}

func TestGetIdentity_ReportsARecordWhoseDeviceIsNotInItsChain(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(k.keys.keys[keyName])
	if err != nil {
		t.Fatal(err)
	}
	record, err := identitylib.ParseRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := identitylib.NewDeviceKey(bytes.NewReader(bytes.Repeat([]byte{9}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	record.Device = stranger
	k.keys.keys[keyName] = base64.RawURLEncoding.EncodeToString(record.Bytes())
	if _, err := k.get.Execute(t.Context()); !errors.Is(err, identitylib.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}

func TestGetIdentity_ReportsAKeyStoreFailure(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	k.keys.getFails = errBroken
	_, err := k.get.Execute(t.Context())
	if !errors.Is(err, errBroken) || errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("error = %v, want the key store failure and not ErrNoIdentity", err)
	}
}

func TestGenerateRecoveryWords_ReturnsTwelveParsableWordsAndStoresNothing(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	first, err := k.generate.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	second, err := k.generate.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 12 {
		t.Fatalf("got %d words, want 12", len(first))
	}
	if _, err := identitylib.ParseRecoveryWords(first); err != nil {
		t.Errorf("the words do not parse: %v", err)
	}
	if slices.Equal(first, second) {
		t.Error("two calls gave the same words")
	}
	if got := k.keys.snapshot(); len(got) != 0 || k.keys.setCalled != 0 {
		t.Errorf("the key store holds %v after generating words", got)
	}
}

func TestCreateIdentity_KeepsOneRecordThatGetIdentityReads(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	created, err := k.create.Execute(t.Context(), getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	stored := k.keys.snapshot()
	if len(stored) != 1 || k.keys.setCalled != 1 {
		t.Fatalf("key store = %v after %d writes, want one entry written once", stored, k.keys.setCalled)
	}
	raw, err := base64.RawURLEncoding.DecodeString(stored[keyName])
	if err != nil {
		t.Fatalf("the record is not base64url without padding: %v", err)
	}
	record, err := identitylib.ParseRecord(raw)
	if err != nil {
		t.Fatalf("ParseRecord() = %v", err)
	}
	if !fingerprint.MatchString(created.AccountFingerprint) || !fingerprint.MatchString(created.DeviceFingerprint) {
		t.Errorf("fingerprints = %q, %q", created.AccountFingerprint, created.DeviceFingerprint)
	}
	if created.AccountFingerprint == created.DeviceFingerprint {
		t.Error("account and device fingerprint are equal")
	}
	if created.DeviceFingerprint != record.Device.Public().Fingerprint().String() {
		t.Error("the device fingerprint is not the one of the device key of the record")
	}
	if created.DeviceName != "Laptop" || !created.DeviceAddedAt.Equal(getNow()) {
		t.Errorf("device = %q added %v", created.DeviceName, created.DeviceAddedAt)
	}
	got, err := k.get.Execute(t.Context())
	if err != nil || got != created {
		t.Errorf("GetIdentity() = %+v, %v; want %+v", got, err, created)
	}
}

func TestCreateIdentity_KeepsNoWordNoEntropyNoRecoverySecret(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	entropy := bytes.Repeat([]byte{0x7f}, 16)
	secret, err := hkdf.Key(sha256.New, entropy, nil, "numen/v1/recovery", 32)
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range k.keys.snapshot() {
		raw, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			t.Fatal(err)
		}
		for _, word := range getLegalWords() {
			if strings.Contains(value, word) || bytes.Contains(raw, []byte(word)) {
				t.Errorf("%s holds the word %q", name, word)
			}
		}
		if bytes.Contains(raw, entropy) || bytes.Contains(raw, secret) {
			t.Errorf("%s holds the entropy or the recovery secret", name)
		}
	}
}

func TestCreateIdentity_RefusesWordsThatAreNotValidAndKeepsNothing(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	abandon := strings.Fields(strings.Repeat("abandon ", 12))
	unknown := slices.Clone(getLegalWords())
	unknown[3] = "numen"
	for name, words := range map[string][]string{"checksum": abandon, "count": getLegalWords()[:11], "unknown word": unknown} {
		_, err := k.create.Execute(t.Context(), words)
		if !errors.Is(err, identitydomain.ErrInvalidWords) {
			t.Fatalf("%s: error = %v, want ErrInvalidWords", name, err)
		}
		if strings.Contains(err.Error(), "numen") {
			t.Errorf("%s: the error echoes a word: %v", name, err)
		}
	}
	if !errors.Is(func() error { _, err := k.create.Execute(t.Context(), abandon); return err }(), identitylib.ErrChecksum) {
		t.Error("ErrInvalidWords does not wrap the error of the library")
	}
	if got := k.keys.snapshot(); len(got) != 0 {
		t.Errorf("the key store holds %v", got)
	}
}

func TestCreateIdentity_NeverOverwritesARecord(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	before := k.keys.snapshot()[keyName]
	other := strings.Fields("letter advice cage absurd amount doctor acoustic avoid letter advice cage above")
	if _, err := k.create.Execute(t.Context(), other); !errors.Is(err, identitydomain.ErrIdentityExists) {
		t.Fatalf("error = %v, want ErrIdentityExists", err)
	}
	if k.keys.snapshot()[keyName] != before || k.keys.setCalled != 1 {
		t.Error("the record changed")
	}
}

func TestCreateIdentity_ReportsAKeyStoreThatRefusesAndKeepsNothing(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	k.keys.setFails = errBroken
	_, err := k.create.Execute(t.Context(), getLegalWords())
	if !errors.Is(err, identitydomain.ErrKeyStore) || !errors.Is(err, errBroken) {
		t.Fatalf("error = %v, want ErrKeyStore wrapping the failure", err)
	}
	if got := k.keys.snapshot(); len(got) != 0 {
		t.Errorf("the key store holds %v", got)
	}
	k.keys.getFails = errBroken
	if _, err := k.create.Execute(t.Context(), getLegalWords()); !errors.Is(err, identitydomain.ErrKeyStore) {
		t.Errorf("a refused read: error = %v, want ErrKeyStore", err)
	}
}

func TestCreateIdentity_TruncatesALongDeviceNameAtACharacter(t *testing.T) {
	t.Parallel()
	k := newKit(strings.Repeat("€", 30))
	created, err := k.create.Execute(t.Context(), getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Repeat("€", 21); created.DeviceName != want {
		t.Errorf("DeviceName = %q, want 21 characters (63 bytes)", created.DeviceName)
	}
}

func TestCreateIdentity_SameWordsOnAnotherDeviceGiveTheSameAccount(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	first, err := k.create.Execute(t.Context(), getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	if err := k.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	second, err := k.create.Execute(t.Context(), getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	if first.AccountFingerprint != second.AccountFingerprint {
		t.Errorf("account fingerprints differ: %s and %s", first.AccountFingerprint, second.AccountFingerprint)
	}
	if first.DeviceFingerprint == second.DeviceFingerprint {
		t.Error("the new identity has the device key of the old one")
	}
	other := newKit("Phone")
	third, err := other.create.Execute(t.Context(), getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	if third.AccountFingerprint != first.AccountFingerprint {
		t.Error("another installation with the same words has another account")
	}
}

func TestDeleteIdentity_RemovesTheRecordAndAMissingRecordIsNotAnError(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	if err := k.delete.Execute(t.Context()); err != nil {
		t.Errorf("delete without a record = %v, want nil", err)
	}
	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	if err := k.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := k.keys.snapshot(); len(got) != 0 {
		t.Errorf("the key store holds %v", got)
	}
	if _, err := k.get.Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("GetIdentity() error = %v, want ErrNoIdentity", err)
	}
}
