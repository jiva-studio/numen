package keychain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/zalando/go-keyring"
)

const (
	entryName  = "secret/s0a1b2c3d"
	otherEntry = "secret/s0e5f6a7b"
	valueOne   = "value-one-for-tests"
	valueTwo   = "value-two-for-tests"
)

func credentialsPath(dir string) string {
	return filepath.Join(dir, "credentials", "credentials.json")
}

func useNoKeychain(t *testing.T) {
	t.Helper()
	keyring.MockInitWithError(fmt.Errorf("the secret service is not running: %w", keyring.ErrUnsupportedPlatform))
}

func readCredentials(t *testing.T, dir string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(credentialsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]string{}
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("the credentials file is not one flat JSON object: %v\n%s", err, raw)
	}
	return entries
}

func TestAValueGoesToTheCredentialsFileWhenTheMachineHasNoKeychain(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	store := newTestStore(dir)

	if err := store.SetKey(t.Context(), entryName, valueOne); err != nil {
		t.Fatal(err)
	}
	if got := readCredentials(t, dir); got[entryName] != valueOne || len(got) != 1 {
		t.Errorf("the credentials file holds %v, want %s under %s", got, valueOne, entryName)
	}
	if got, err := store.GetKey(t.Context(), entryName); err != nil || got != valueOne {
		t.Errorf("read %q, %v", got, err)
	}
}

func TestTheCredentialsFileIsPrivateToTheUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mode bits do not apply")
	}
	useNoKeychain(t)
	dir := t.TempDir()
	if err := newTestStore(dir).SetKey(t.Context(), entryName, valueOne); err != nil {
		t.Fatal(err)
	}
	file, err := os.Stat(credentialsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	folder, err := os.Stat(filepath.Dir(credentialsPath(dir)))
	if err != nil {
		t.Fatal(err)
	}
	if file.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v, want 0600", file.Mode().Perm())
	}
	if folder.Mode().Perm() != 0o700 {
		t.Errorf("directory mode = %v, want 0700", folder.Mode().Perm())
	}
}

func TestEachWriteReplacesTheCredentialsFileThroughARename(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	store := newTestStore(dir)
	if err := store.SetKey(t.Context(), entryName, valueOne); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(credentialsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetKey(t.Context(), otherEntry, valueTwo); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(credentialsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Error("the file was rewritten in place, so a crash can leave half of it")
	}
	entries, err := os.ReadDir(filepath.Dir(credentialsPath(dir)))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("the directory holds %d entries after the writes, want only credentials.json", len(entries))
	}
	if got := readCredentials(t, dir); got[entryName] != valueOne || got[otherEntry] != valueTwo {
		t.Errorf("the file holds %v, want both entries", got)
	}
}

func TestADeleteRemovesTheEntryFromTheCredentialsFileAndKeepsTheOthers(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	store := newTestStore(dir)
	for name, value := range map[string]string{entryName: valueOne, otherEntry: valueTwo} {
		if err := store.SetKey(t.Context(), name, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.DeleteKey(t.Context(), entryName); err != nil {
		t.Fatal(err)
	}
	got := readCredentials(t, dir)
	if _, found := got[entryName]; found || got[otherEntry] != valueTwo {
		t.Errorf("the file holds %v", got)
	}
	if _, err := store.GetKey(t.Context(), entryName); !errors.Is(err, plugin.ErrNoKey) {
		t.Errorf("a deleted value read as %v, want ErrNoKey", err)
	}
}

func TestAKeychainThatAppearsLaterIsUsedAndTheFileStillAnswersForItsKeys(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	store := newTestStore(dir)
	if err := store.SetKey(t.Context(), entryName, valueOne); err != nil {
		t.Fatal(err)
	}

	keyring.MockInit()
	if err := store.SetKey(t.Context(), otherEntry, valueTwo); err != nil {
		t.Fatal(err)
	}
	if got, err := keyring.Get(service, otherEntry); err != nil || got != valueTwo {
		t.Errorf("a write after the keychain appeared did not reach it: %q, %v", got, err)
	}
	if _, err := store.GetKey(t.Context(), entryName); !errors.Is(err, plugin.ErrNoKey) {
		t.Errorf("a key kept in the file read as %v while the keychain is reachable, want ErrNoKey", err)
	}
}

func TestAKeyStoredInTheKeychainIsRemovedFromTheFile(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	store := newTestStore(dir)
	for name, value := range map[string]string{entryName: valueOne, otherEntry: valueTwo} {
		if err := store.SetKey(t.Context(), name, value); err != nil {
			t.Fatal(err)
		}
	}

	keyring.MockInit()
	other := newTestStore(dir)
	if err := other.SetKey(t.Context(), entryName, "rotated-value"); err != nil {
		t.Fatal(err)
	}
	got := readCredentials(t, dir)
	if _, found := got[entryName]; found || got[otherEntry] != valueTwo {
		t.Errorf("the file holds %v after the key moved to the keychain, want only %s", got, otherEntry)
	}
	if value, err := keyring.Get(service, entryName); err != nil || value != "rotated-value" {
		t.Errorf("the keychain holds %q, %v", value, err)
	}
}

func TestAKeychainThatRefusesIsNotReplacedByTheFile(t *testing.T) {
	useReachableSessionBus(t)
	keyring.MockInitWithError(errors.New("the collection is locked"))
	dir := t.TempDir()
	store := newTestStore(dir)

	err := store.SetKey(t.Context(), entryName, valueOne)
	if !errors.Is(err, plugin.ErrNoKeyStore) {
		t.Fatalf("a write to a refusing keychain returned %v, want ErrNoKeyStore", err)
	}
	if _, statErr := os.Stat(credentialsPath(dir)); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("a key was written to the file: %v", statErr)
	}
	if got := store.GetCredentialsPath(t.Context()); got != "" {
		t.Errorf("path = %q for a keychain that refuses, want none", got)
	}
}

func TestWithAReachableKeychainTheCredentialsFileIsNotCreated(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	store := newTestStore(dir)

	if err := store.SetKey(t.Context(), entryName, valueOne); err != nil {
		t.Fatal(err)
	}
	if got, err := store.GetKey(t.Context(), entryName); err != nil || got != valueOne {
		t.Fatalf("read %q, %v", got, err)
	}
	if err := store.DeleteKey(t.Context(), entryName); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "credentials")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the credentials directory exists with a working keychain: %v", err)
	}
}

func TestTheCredentialsFileIsNotReadWhileTheKeychainIsReachable(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(credentialsPath(dir)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credentialsPath(dir), []byte(`{"`+entryName+`":"`+valueOne+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := newTestStore(dir)

	if _, err := store.GetKey(t.Context(), entryName); !errors.Is(err, plugin.ErrNoKey) {
		t.Errorf("error = %v, want ErrNoKey: the file is not read while the keychain is reachable", err)
	}
	if err := keyring.Set(service, entryName, valueTwo); err != nil {
		t.Fatal(err)
	}
	if got, err := store.GetKey(t.Context(), entryName); err != nil || got != valueTwo {
		t.Errorf("read %q, %v, want the value of the keychain", got, err)
	}
}

func TestAValueWithNoEntryAnywhereIsNoKey(t *testing.T) {
	keyring.MockInit()
	if _, err := newTestStore(t.TempDir()).GetKey(t.Context(), entryName); !errors.Is(err, plugin.ErrNoKey) {
		t.Errorf("error = %v, want ErrNoKey", err)
	}
}

func TestAValueOverTheSmallestKeychainLimitFailsNamingTheLimitAndStoresNothing(t *testing.T) {
	tooLong := strings.Repeat("v", 2561)
	atLimit := strings.Repeat("v", 2560)

	t.Run("keychain", func(t *testing.T) {
		keyring.MockInit()
		store := Store{}
		err := store.SetKey(t.Context(), entryName, tooLong)
		if err == nil || !strings.Contains(err.Error(), "2560") || strings.Contains(err.Error(), tooLong) {
			t.Fatalf("error = %v, want one that names 2560 and not the value", err)
		}
		if _, err := keyring.Get(service, entryName); !errors.Is(err, keyring.ErrNotFound) {
			t.Errorf("a value over the limit was stored: %v", err)
		}
		if err := store.SetKey(t.Context(), entryName, atLimit); err != nil {
			t.Errorf("a value of exactly 2560 bytes: %v", err)
		}
	})
	t.Run("file", func(t *testing.T) {
		useNoKeychain(t)
		dir := t.TempDir()
		store := newTestStore(dir)
		err := store.SetKey(t.Context(), entryName, tooLong)
		if err == nil || !strings.Contains(err.Error(), "2560") || strings.Contains(err.Error(), tooLong) {
			t.Fatalf("error = %v, want one that names 2560 and not the value", err)
		}
		if raw, err := os.ReadFile(credentialsPath(dir)); err == nil && strings.Contains(string(raw), tooLong) {
			t.Error("a value over the limit was written to the file")
		}
		if err := store.SetKey(t.Context(), entryName, atLimit); err != nil {
			t.Errorf("a value of exactly 2560 bytes: %v", err)
		}
	})
}

func TestNoErrorOfAFailingStoreCarriesTheValue(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	// The credentials directory is a file, so the fallback cannot be written either.
	if err := os.WriteFile(filepath.Join(dir, "credentials"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := newTestStore(dir).SetKey(t.Context(), entryName, valueOne)
	if err == nil {
		t.Fatal("a write with no place to go succeeded")
	}
	if strings.Contains(err.Error(), valueOne) {
		t.Errorf("the error carries the value: %v", err)
	}
}
