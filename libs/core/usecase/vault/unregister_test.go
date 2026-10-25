package vault_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// indexRows is the index as a vault is written to and taken out of it: a set
// of identifiers, each either registered or not. A vault already registered
// keeps the row it has.
type indexRows struct {
	saved        []domain.VaultID
	rows         map[domain.VaultID]bool
	unregistered []domain.VaultID
	fails        error
	steps        *[]string
}

func (r *indexRows) Register(_ context.Context, id domain.VaultID) error {
	if r.fails != nil {
		return r.fails
	}
	if !r.rows[id] {
		r.saved = append(r.saved, id)
		if r.rows == nil {
			r.rows = map[domain.VaultID]bool{}
		}
		r.rows[id] = true
	}
	return nil
}

func (r *indexRows) Unregister(_ context.Context, vaultID domain.VaultID) error {
	r.unregistered = append(r.unregistered, vaultID)
	if r.steps != nil {
		*r.steps = append(*r.steps, "forget")
	}
	if r.fails != nil {
		return r.fails
	}
	delete(r.rows, vaultID)
	return nil
}

// twoVaults is two folders this installation knows about, and the registry it
// knows them through.
func twoVaults(t *testing.T) (domain.Vault, domain.Vault, *appstate.VaultRegistry) {
	t.Helper()
	add, registry := newAdd(t)
	first, err := add.Execute(folder(t, "personal"), "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := add.Execute(folder(t, "work"), "")
	if err != nil {
		t.Fatal(err)
	}
	return first, second, registry
}

func TestForgetConsumesTheVaultOffTheListAndOutOfTheIndex(t *testing.T) {
	t.Parallel()
	retained, removed, registry := twoVaults(t)
	index := &indexRows{}

	if err := vaults.NewUnregister(registry, index).Execute(t.Context(), removed); err != nil {
		t.Fatal(err)
	}

	recognized, err := registry.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 1 || recognized[0].ID != retained.ID {
		t.Errorf("the list holds %v, want only %s", recognized, retained.Name)
	}
	if len(index.unregistered) != 1 || index.unregistered[0] != removed.ID {
		t.Errorf("forgot %v, want %s", index.unregistered, removed.ID)
	}
}

func TestForgetPreservesTheFolderWhereItIs(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)

	if err := vaults.NewUnregister(registry, &indexRows{}).Execute(t.Context(), removed); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(removed.Path, ".numen", "config.json")); err != nil {
		t.Errorf("the folder no longer carries its identity: %v", err)
	}
}

func TestForgetRefusesTheOnlyVault(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	only, err := add.Execute(folder(t, "personal"), "")
	if err != nil {
		t.Fatal(err)
	}
	index := &indexRows{}

	err = vaults.NewUnregister(registry, index).Execute(t.Context(), only)
	if !errors.Is(err, vaults.ErrLastVault) {
		t.Fatalf("the last vault was answered %v", err)
	}
	recognized, err := registry.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 1 {
		t.Errorf("the list holds %v, want the vault still on it", recognized)
	}
	if len(index.unregistered) != 0 {
		t.Errorf("forgot %v", index.unregistered)
	}
}

func TestAVaultTheIndexCouldNotForgetStaysOnTheList(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)
	index := &indexRows{fails: errors.New("the index is locked")}

	if err := vaults.NewUnregister(registry, index).Execute(t.Context(), removed); err == nil {
		t.Fatal("an index that failed was reported as success")
	}
	if _, found, err := registry.FindVault(string(removed.ID)); err != nil || !found {
		t.Errorf("the vault left the list with its rows still in the index: %v %v", found, err)
	}
}

func TestForgetKeepsTheCauseWhenTheIndexRefuses(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)
	index := &indexRows{fails: errRefused}

	if err := vaults.NewUnregister(registry, index).Execute(t.Context(), removed); !errors.Is(err, errRefused) {
		t.Fatalf("want the index's error inside the error, got %v", err)
	}
}

func TestForgetStopsWhenTheListCannotBeRead(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)
	index := &indexRows{}

	err := vaults.NewUnregister(refusingRegistry{VaultRegistry: registry, list: errRefused}, index).Execute(t.Context(), removed)
	if !errors.Is(err, errRefused) {
		t.Fatalf("want the rejection back, got %v", err)
	}
	if len(index.unregistered) != 0 {
		t.Errorf("forgot %v", index.unregistered)
	}
}

func TestEitherVaultOfTwoMayBeForgotten(t *testing.T) {
	t.Parallel()
	first, second, registry := twoVaults(t)

	if err := vaults.NewUnregister(registry, &indexRows{}).Execute(t.Context(), first); err != nil {
		t.Fatalf("the vault listed first failed: %v", err)
	}
	recognized, err := registry.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 1 || recognized[0].ID != second.ID {
		t.Errorf("the list holds %v, want only %s", recognized, second.Name)
	}
}

// A vault the list no longer holds is not the last one, whatever else is left.
func TestAVaultNotOnTheListIsNotTheLastVault(t *testing.T) {
	t.Parallel()
	first, _, registry := twoVaults(t)
	if err := registry.Remove(first.ID); err != nil {
		t.Fatal(err)
	}

	if err := vaults.NewUnregister(registry, &indexRows{}).Execute(t.Context(), first); err != nil {
		t.Fatalf("a vault that is not on the list failed: %v", err)
	}
	recognized, err := registry.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 1 {
		t.Errorf("the list holds %v, want the other vault still on it", recognized)
	}
}
