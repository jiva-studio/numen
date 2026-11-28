package keyvault_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/apps/mobile/internal/keyvault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// memoryVault is the vault of keys of the platform, in memory.
type memoryVault struct{ keys map[string]string }

func newMemoryVault() *memoryVault { return &memoryVault{keys: map[string]string{}} }

func (v *memoryVault) Get(name string) (string, error) { return v.keys[name], nil }

func (v *memoryVault) Has(name string) bool {
	_, isThere := v.keys[name]
	return isThere
}

func (v *memoryVault) Set(name, value string) error {
	v.keys[name] = value
	return nil
}

func (v *memoryVault) Delete(name string) error {
	delete(v.keys, name)
	return nil
}

func TestGetKeyReturnsErrNoKeyForAName(t *testing.T) {
	if _, err := keyvault.New(newMemoryVault()).GetKey(t.Context(), "identity"); !errors.Is(err, plugin.ErrNoKey) {
		t.Fatalf("got %v, want ErrNoKey", err)
	}
}

func TestGetKeyReturnsWhatSetKeyStored(t *testing.T) {
	store := keyvault.New(newMemoryVault())
	if err := store.SetKey(t.Context(), "identity", "secret-1"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetKey(t.Context(), "identity", "secret-2"); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetKey(t.Context(), "identity")
	if err != nil || got != "secret-2" {
		t.Fatalf("got %q, %v, want the second key", got, err)
	}
}

func TestKeysAreStoredByTheirOwnNames(t *testing.T) {
	store := keyvault.New(newMemoryVault())
	if err := store.SetKey(t.Context(), "identity", "one"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetKey(t.Context(), "provider"); !errors.Is(err, plugin.ErrNoKey) {
		t.Fatalf("got %v, want ErrNoKey for a name nothing was stored under", err)
	}
}

func TestDeleteKeyRemovesTheKey(t *testing.T) {
	store := keyvault.New(newMemoryVault())
	if err := store.SetKey(t.Context(), "identity", "one"); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteKey(t.Context(), "identity"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetKey(t.Context(), "identity"); !errors.Is(err, plugin.ErrNoKey) {
		t.Fatalf("got %v, want ErrNoKey after the delete", err)
	}
}

func TestDeleteKeyOfAMissingNameReturnsErrNoKey(t *testing.T) {
	if err := keyvault.New(newMemoryVault()).DeleteKey(t.Context(), "identity"); !errors.Is(err, plugin.ErrNoKey) {
		t.Fatalf("got %v, want ErrNoKey", err)
	}
}

func TestSetKeyAcceptsAKeyOfTheLimitAndRefusesALongerOne(t *testing.T) {
	store := keyvault.New(newMemoryVault())
	atLimit := strings.Repeat("v", keyvault.MaxValueBytes)
	if err := store.SetKey(t.Context(), "identity", atLimit); err != nil {
		t.Fatalf("a key of %d bytes was refused: %v", keyvault.MaxValueBytes, err)
	}
	tooLong := atLimit + "v"
	err := store.SetKey(t.Context(), "provider", tooLong)
	if err == nil || !strings.Contains(err.Error(), "2560") || strings.Contains(err.Error(), tooLong) {
		t.Fatalf("got %v, want an error that names 2560 and not the value", err)
	}
	if _, err := store.GetKey(t.Context(), "provider"); !errors.Is(err, plugin.ErrNoKey) {
		t.Fatalf("got %v, want nothing stored for a key that was refused", err)
	}
}
