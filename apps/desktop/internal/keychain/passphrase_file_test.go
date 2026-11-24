package keychain

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zalando/go-keyring"
)

const (
	seedValue     = "c2VlZC1mb3ItdGVzdHMtb25seS0zMmJ5dGVzLXNlZWQ="
	passphraseOne = "correct horse battery staple"
	passphraseTwo = "another passphrase entirely"
)

type fakeSource struct {
	passphrase string
	calls      atomic.Int32
}

func (f *fakeSource) GetPassphrase(context.Context) (string, error) {
	f.calls.Add(1)
	return f.passphrase, nil
}

func newPassphraseStore(dir string, source PassphraseSource) *Store {
	store := newTestStore(dir)
	store.kdf = kdfParams{memoryKiB: minKDFMemoryKiB, time: 1, threads: 1}
	store.UsePassphraseFile(source)
	return store
}

func TestSetKeyFailsBeforeAFileModeIsChosen(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	store := NewStore(dir, waitOnTimer)

	err := store.SetKey(t.Context(), entryName, seedValue)

	if !errors.Is(err, ErrFileBackendNotConfirmed) {
		t.Fatalf("SetKey = %v, want ErrFileBackendNotConfirmed", err)
	}
	if _, statErr := os.Stat(credentialsPath(dir)); !os.IsNotExist(statErr) {
		t.Errorf("the credentials file exists after a refused write: %v", statErr)
	}
}

func TestAPassphraseFileIsReadByANewStore(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	if err := newPassphraseStore(dir, &fakeSource{passphrase: passphraseOne}).SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}

	got, err := newPassphraseStore(dir, &fakeSource{passphrase: passphraseOne}).GetKey(t.Context(), entryName)

	if err != nil || got != seedValue {
		t.Errorf("GetKey = %q, %v, want the seed", got, err)
	}
}

func TestTheEncryptedFileContainsNeitherTheSeedNorTheEntryName(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	if err := newPassphraseStore(dir, &fakeSource{passphrase: passphraseOne}).SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(credentialsPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(raw), seedValue) || strings.Contains(string(raw), entryName) {
		t.Errorf("the encrypted file holds plaintext: %q", raw)
	}
	if !isEncryptedFile(raw) {
		t.Error("the file is not in the encrypted format")
	}
}

func TestAWrongPassphraseDoesNotOpenTheFile(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	if err := newPassphraseStore(dir, &fakeSource{passphrase: passphraseOne}).SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}

	got, err := newPassphraseStore(dir, &fakeSource{passphrase: passphraseTwo}).GetKey(t.Context(), entryName)

	if !errors.Is(err, ErrWrongPassphrase) || got != "" {
		t.Errorf("GetKey = %q, %v, want ErrWrongPassphrase", got, err)
	}
}

func TestAChangedHeaderDoesNotOpenTheFile(t *testing.T) {
	entries := map[string]string{entryName: seedValue}
	params := kdfParams{memoryKiB: minKDFMemoryKiB, time: 1, threads: 1}
	for name, offset := range map[string]int{"the salt": headerSize - 1, "the memory cost": len(encryptedMagic) + 4, "the time cost": len(encryptedMagic) + 8} {
		t.Run(name, func(t *testing.T) {
			raw, err := encryptEntries(entries, passphraseOne, params)
			if err != nil {
				t.Fatal(err)
			}
			raw[offset] ^= 0x01

			if _, err := decryptEntries(raw, passphraseOne); err == nil {
				t.Error("a file with a changed header was opened")
			}
		})
	}
}

func TestAChangedVersionIsNotAnEncryptedFile(t *testing.T) {
	raw, err := encryptEntries(map[string]string{entryName: seedValue}, passphraseOne, kdfParams{memoryKiB: minKDFMemoryKiB, time: 1, threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	raw[len(encryptedMagic)] = encryptedVersion + 1

	if _, err := decryptEntries(raw, passphraseOne); !errors.Is(err, errBadEncryptedFile) {
		t.Errorf("decryptEntries = %v, want errBadEncryptedFile", err)
	}
}

func TestAChangedBodyDoesNotOpenTheFile(t *testing.T) {
	raw, err := encryptEntries(map[string]string{entryName: seedValue}, passphraseOne, kdfParams{memoryKiB: minKDFMemoryKiB, time: 1, threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-1] ^= 0x01

	if _, err := decryptEntries(raw, passphraseOne); !errors.Is(err, ErrWrongPassphrase) {
		t.Errorf("decryptEntries = %v, want ErrWrongPassphrase", err)
	}
}

func TestTheEncryptedHeaderHasTheDefaultCost(t *testing.T) {
	raw, err := encryptEntries(map[string]string{}, passphraseOne, getDefaultKDFParams())
	if err != nil {
		t.Fatal(err)
	}

	params, _, err := readHeader(raw[:headerSize])

	if err != nil || params.memoryKiB < 64*1024 || params.time != 3 || params.threads != 1 {
		t.Errorf("header params = %+v, %v, want at least 64 MiB, t=3, p=1", params, err)
	}
}

func TestThePassphraseIsAskedOnceForManyOperations(t *testing.T) {
	useNoKeychain(t)
	source := &fakeSource{passphrase: passphraseOne}
	store := newPassphraseStore(t.TempDir(), source)

	for _, name := range []string{entryName, otherEntry} {
		if err := store.SetKey(t.Context(), name, seedValue); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.GetKey(t.Context(), entryName); err != nil {
		t.Fatal(err)
	}

	if got := source.calls.Load(); got != 1 {
		t.Errorf("the passphrase was asked %d times, want 1", got)
	}
}

func TestAPlaintextFileIsEncryptedByTheNextWriteInPassphraseMode(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	plain := newTestStore(dir)
	if err := plain.SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}
	store := newPassphraseStore(dir, &fakeSource{passphrase: passphraseOne})

	if err := store.SetKey(t.Context(), otherEntry, valueOne); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(credentialsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), seedValue) {
		t.Error("the old plaintext entry is still in the file")
	}
	if got, err := store.GetKey(t.Context(), entryName); err != nil || got != seedValue {
		t.Errorf("GetKey = %q, %v, want the old entry", got, err)
	}
}

func TestAnEncryptedFileIsNotReadBeforeAPassphraseIsChosen(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	if err := newPassphraseStore(dir, &fakeSource{passphrase: passphraseOne}).SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}

	_, err := NewStore(dir, waitOnTimer).GetKey(t.Context(), entryName)

	if !errors.Is(err, ErrFileBackendNotConfirmed) {
		t.Errorf("GetKey = %v, want ErrFileBackendNotConfirmed", err)
	}
}

func TestTheStateWarnsWhileKeysAreNotProtectedByAPassphrase(t *testing.T) {
	useNoKeychain(t)
	for name, test := range map[string]struct {
		prepare     func(*Store)
		wantWarning bool
		wantMode    FileMode
	}{
		"no choice yet":  {func(*Store) {}, true, FileModeUnset},
		"plaintext file": {func(s *Store) { s.UsePlaintextFile() }, true, FileModePlaintext},
		"passphrase":     {func(s *Store) { s.UsePassphraseFile(&fakeSource{passphrase: passphraseOne}) }, false, FileModePassphrase},
	} {
		t.Run(name, func(t *testing.T) {
			store := NewStore(t.TempDir(), waitOnTimer)
			test.prepare(store)

			state, err := store.GetFileState(t.Context())

			if err != nil || !state.IsInUse || state.Mode != test.wantMode || (state.Warning != "") != test.wantWarning {
				t.Errorf("state = %+v, %v, want in use, mode %d, warning %v", state, err, test.wantMode, test.wantWarning)
			}
		})
	}
}

func TestTheStateHasNoWarningWhileTheKeychainIsReachable(t *testing.T) {
	keyring.MockInit()
	store := NewStore(t.TempDir(), waitOnTimer)

	state, err := store.GetFileState(t.Context())

	if err != nil || state != (FileState{}) {
		t.Errorf("state = %+v, %v, want the zero state", state, err)
	}
}

func TestAPlaintextFileWritesOneWarningToTheLog(t *testing.T) {
	useNoKeychain(t)
	var warnings []string
	store := newTestStore(t.TempDir())
	store.warn = func(text string) { warnings = append(warnings, text) }

	for _, name := range []string{entryName, otherEntry} {
		if err := store.SetKey(t.Context(), name, valueOne); err != nil {
			t.Fatal(err)
		}
	}

	if len(warnings) != 1 || warnings[0] != warningPlaintext {
		t.Errorf("warnings = %q, want one plaintext warning", warnings)
	}
}

type failingSource struct{ err error }

func (f failingSource) GetPassphrase(context.Context) (string, error) {
	return "", f.err
}

func TestACancelledPassphrasePromptLeavesAPlaintextFileUnchangedOnSetKey(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	if err := newTestStore(dir).SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}
	before := readCredentialsBytes(t, dir)
	cancelled := errors.New("the prompt was cancelled")
	store := newPassphraseStore(dir, failingSource{err: cancelled})

	err := store.SetKey(t.Context(), otherEntry, valueOne)

	if !errors.Is(err, cancelled) {
		t.Fatalf("SetKey = %v, want the prompt error", err)
	}
	if after := readCredentialsBytes(t, dir); string(after) != string(before) {
		t.Errorf("the credentials file changed: %q, want %q", after, before)
	}
}

func TestACancelledPassphrasePromptLeavesAPlaintextFileUnchangedOnDeleteKey(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	plain := newTestStore(dir)
	for name, value := range map[string]string{entryName: seedValue, otherEntry: valueOne} {
		if err := plain.SetKey(t.Context(), name, value); err != nil {
			t.Fatal(err)
		}
	}
	before := readCredentialsBytes(t, dir)
	cancelled := errors.New("the prompt was cancelled")
	store := newPassphraseStore(dir, failingSource{err: cancelled})

	err := store.DeleteKey(t.Context(), otherEntry)

	if !errors.Is(err, cancelled) {
		t.Fatalf("DeleteKey = %v, want the prompt error", err)
	}
	if after := readCredentialsBytes(t, dir); string(after) != string(before) {
		t.Errorf("the credentials file changed: %q, want %q", after, before)
	}
}

func TestACancelledPassphrasePromptCreatesNoFileOnAFirstWrite(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	cancelled := errors.New("the prompt was cancelled")
	store := newPassphraseStore(dir, failingSource{err: cancelled})

	err := store.SetKey(t.Context(), entryName, seedValue)

	if !errors.Is(err, cancelled) {
		t.Fatalf("SetKey = %v, want the prompt error", err)
	}
	if _, statErr := os.Stat(credentialsPath(dir)); !os.IsNotExist(statErr) {
		t.Errorf("the credentials file exists after a failed first write: %v", statErr)
	}
}

func readCredentialsBytes(t *testing.T, dir string) []byte {
	t.Helper()
	raw, err := os.ReadFile(credentialsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
