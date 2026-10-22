package vault_test

import (
	"errors"
	"os"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestARenamedVaultKeepsItsNameOnTheListAndItsRowInTheIndex(t *testing.T) {
	t.Parallel()
	_, renamed, registry := twoVaults(t)
	index := &indexRows{}

	got, err := vaults.NewRename(registry, index).Execute(t.Context(), renamed, "journal")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "journal" {
		t.Errorf("the vault is called %q", got.Name)
	}

	onTheList, found, err := registry.FindVault(string(renamed.ID))
	if err != nil || !found {
		t.Fatalf("the vault left the list: %v %v", found, err)
	}
	if onTheList.Name != "journal" {
		t.Errorf("the list calls it %q", onTheList.Name)
	}
	if len(index.saved) != 1 || index.saved[0] != got.ID {
		t.Errorf("the index received about %v, want %s", index.saved, got.ID)
	}
	if _, err := os.Stat(renamed.Path); err != nil {
		t.Errorf("the folder was renamed with the vault: %v", err)
	}
}

func TestANameAnotherVaultHasIsRefused(t *testing.T) {
	t.Parallel()
	acquired, renamed, registry := twoVaults(t)
	index := &indexRows{}

	// The comparison is the one the list is searched by: without case, over
	// normalised text.
	_, err := vaults.NewRename(registry, index).Execute(t.Context(), renamed, "PERSONAL")
	if !errors.Is(err, vaults.ErrNameTaken) {
		t.Fatalf("a name %s already has was answered %v", acquired.Name, err)
	}
	onTheList, found, err := registry.FindVault(string(renamed.ID))
	if err != nil || !found {
		t.Fatalf("the vault left the list: %v %v", found, err)
	}
	if onTheList.Name != renamed.Name {
		t.Errorf("the list calls it %q, want %q", onTheList.Name, renamed.Name)
	}
	if len(index.saved) != 0 {
		t.Errorf("the index received %v", index.saved)
	}
}

func TestTheNameAVaultAlreadyHasChangesNothing(t *testing.T) {
	t.Parallel()
	_, v, registry := twoVaults(t)
	index := &indexRows{}

	got, err := vaults.NewRename(registry, index).Execute(t.Context(), v, v.Name)
	if err != nil {
		t.Fatalf("renaming a vault to what it is called: %v", err)
	}
	if got != v {
		t.Errorf("got %v, want %v", got, v)
	}
	if len(index.saved) != 0 {
		t.Errorf("the index was written to: %v", index.saved)
	}
}

// faultyRegistry is a list that fails when it is read or written.
type faultyRegistry struct {
	port.VaultRegistry
	listErr error
	saveErr error
}

func (r faultyRegistry) List() ([]domain.Vault, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.VaultRegistry.List()
}

func (r faultyRegistry) Save(v domain.Vault) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	return r.VaultRegistry.Save(v)
}

var errBroken = errors.New("broken")

func TestAVaultIsNotRenamedToNothing(t *testing.T) {
	t.Parallel()
	_, v, registry := twoVaults(t)
	index := &indexRows{}

	if _, err := vaults.NewRename(registry, index).Execute(t.Context(), v, ""); err == nil {
		t.Fatal("an empty name was accepted")
	}
	onTheList, _, err := registry.FindVault(string(v.ID))
	if err != nil {
		t.Fatal(err)
	}
	if onTheList.Name != v.Name {
		t.Errorf("the list calls it %q, want %q", onTheList.Name, v.Name)
	}
	if len(index.saved) != 0 {
		t.Errorf("the index received %v", index.saved)
	}
}

func TestARenameFailsWhenTheListCannotBeRead(t *testing.T) {
	t.Parallel()
	_, v, registry := twoVaults(t)
	index := &indexRows{}

	_, err := vaults.NewRename(faultyRegistry{VaultRegistry: registry, listErr: errBroken}, index).
		Execute(t.Context(), v, "journal")
	if !errors.Is(err, errBroken) {
		t.Fatalf("got %v, want %v", err, errBroken)
	}
	if len(index.saved) != 0 {
		t.Errorf("the index received %v", index.saved)
	}
}

func TestARenameFailsWhenTheListCannotBeSaved(t *testing.T) {
	t.Parallel()
	_, v, registry := twoVaults(t)
	index := &indexRows{}

	_, err := vaults.NewRename(faultyRegistry{VaultRegistry: registry, saveErr: errBroken}, index).
		Execute(t.Context(), v, "journal")
	if !errors.Is(err, errBroken) {
		t.Fatalf("got %v, want %v", err, errBroken)
	}
	if len(index.saved) != 0 {
		t.Errorf("the index received %v", index.saved)
	}
}

func TestARenameFailsWhenTheIndexRefusesTheVault(t *testing.T) {
	t.Parallel()
	_, v, registry := twoVaults(t)

	_, err := vaults.NewRename(registry, &indexRows{fails: errBroken}).Execute(t.Context(), v, "journal")
	if !errors.Is(err, errBroken) {
		t.Fatalf("got %v, want %v", err, errBroken)
	}
}
