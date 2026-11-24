package keychain

import (
	"errors"
	"os"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

func newKeyFile(dir string) (*KeyFile, *Store) {
	store := NewStore(dir, waitOnTimer)
	store.kdf = kdfParams{memoryKiB: minKDFMemoryKiB, time: 1, threads: 1}
	return NewKeyFile(store, &Passphrase{}), store
}

func TestPassphraseWaitsForTheWindow(t *testing.T) {
	var passphrase Passphrase

	if _, err := passphrase.GetPassphrase(t.Context()); !errors.Is(err, ErrPassphraseNeeded) {
		t.Fatalf("before Set: %v", err)
	}
	passphrase.Set(passphraseOne)

	got, err := passphrase.GetPassphrase(t.Context())
	if err != nil || got != passphraseOne {
		t.Fatalf("after Set: %q, %v", got, err)
	}
}

func TestKeyFileAsksForAChoiceWhereThereIsNoKeychain(t *testing.T) {
	useNoKeychain(t)
	file, _ := newKeyFile(t.TempDir())

	state, err := file.GetState(t.Context())

	if err != nil || !state.IsInUse || state.Mode != pluginport.KeyFileUnset || state.Warning == "" {
		t.Fatalf("state %+v, err %v", state, err)
	}
}

func TestKeyFileKeepsKeysEncryptedAfterAPassphraseChoice(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	file, store := newKeyFile(dir)

	if err := file.UsePassphrase(t.Context(), passphraseOne); err != nil {
		t.Fatal(err)
	}
	if err := store.SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(credentialsPath(dir))
	if err != nil || !isEncryptedFile(raw) {
		t.Fatalf("file encrypted %v, err %v", isEncryptedFile(raw), err)
	}
	state, _ := file.GetState(t.Context())
	if state.Mode != pluginport.KeyFilePassphrase || state.IsLocked {
		t.Fatalf("state %+v", state)
	}
}

func TestKeyFileIsLockedAtTheNextStartUntilTheRightPassphraseIsGiven(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	first, store := newKeyFile(dir)
	if err := first.UsePassphrase(t.Context(), passphraseOne); err != nil {
		t.Fatal(err)
	}
	if err := store.SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}
	second, secondStore := newKeyFile(dir)
	secondStore.UsePassphraseFile(second.passphrase)

	locked, _ := second.GetState(t.Context())
	_, readErr := secondStore.GetKey(t.Context(), entryName)
	wrong := second.Unlock(t.Context(), passphraseTwo)
	right := second.Unlock(t.Context(), passphraseOne)
	unlocked, _ := second.GetState(t.Context())
	got, getErr := secondStore.GetKey(t.Context(), entryName)

	if !locked.IsLocked || !errors.Is(readErr, ErrPassphraseNeeded) {
		t.Fatalf("locked %+v, read error %v", locked, readErr)
	}
	if !errors.Is(wrong, pluginport.ErrWrongPassphrase) || right != nil || unlocked.IsLocked {
		t.Fatalf("wrong %v, right %v, unlocked %+v", wrong, right, unlocked)
	}
	if getErr != nil || got != seedValue {
		t.Fatalf("key %q, err %v", got, getErr)
	}
}

func TestKeyFileRefusesAPassphraseThatDoesNotOpenTheFile(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	first, store := newKeyFile(dir)
	if err := first.UsePassphrase(t.Context(), passphraseOne); err != nil {
		t.Fatal(err)
	}
	if err := store.SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}
	second, _ := newKeyFile(dir)

	err := second.UsePassphrase(t.Context(), passphraseTwo)

	if !errors.Is(err, pluginport.ErrWrongPassphrase) {
		t.Fatalf("err %v", err)
	}
}

func TestKeyFileKeepsKeysAsTextAfterAPlaintextChoice(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	file, store := newKeyFile(dir)
	if err := file.UsePlaintext(t.Context()); err != nil {
		t.Fatal(err)
	}

	if err := store.SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(credentialsPath(dir))
	state, _ := file.GetState(t.Context())
	if isEncryptedFile(raw) || state.Mode != pluginport.KeyFilePlaintext || state.Warning == "" {
		t.Fatalf("encrypted %v, state %+v", isEncryptedFile(raw), state)
	}
}

func TestKeyFileWritesTheKeysAgainWhenTheModeChanges(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	file, store := newKeyFile(dir)
	if err := file.UsePassphrase(t.Context(), passphraseOne); err != nil {
		t.Fatal(err)
	}
	if err := store.SetKey(t.Context(), entryName, seedValue); err != nil {
		t.Fatal(err)
	}

	if err := file.UsePlaintext(t.Context()); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(credentialsPath(dir))
	got, err := store.GetKey(t.Context(), entryName)
	if isEncryptedFile(raw) || err != nil || got != seedValue {
		t.Fatalf("encrypted %v, key %q, err %v", isEncryptedFile(raw), got, err)
	}
	if err := file.UsePassphrase(t.Context(), passphraseTwo); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(credentialsPath(dir))
	if !isEncryptedFile(raw) {
		t.Fatal("the keys stayed as plaintext after the passphrase was chosen")
	}
}
