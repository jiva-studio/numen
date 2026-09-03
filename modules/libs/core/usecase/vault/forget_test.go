package vault_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// indexRows is the index as a vault is written to and taken out of it. `rows`
// answers the way the table does: Save writes the row whole, and a vault the
// index already holds keeps its name and its path through Register.
type indexRows struct {
	saved  []domain.Vault
	rows   map[string]domain.Vault
	forgot []string
	fails  error
	steps  *[]string
}

func (r *indexRows) Save(_ context.Context, v domain.Vault) error {
	r.saved = append(r.saved, v)
	if r.fails != nil {
		return r.fails
	}
	r.put(v)
	return nil
}

func (r *indexRows) Register(_ context.Context, v domain.Vault) error {
	if r.fails != nil {
		return r.fails
	}
	if _, there := r.rows[v.ID]; !there {
		r.put(v)
	}
	return nil
}

func (r *indexRows) put(v domain.Vault) {
	if r.rows == nil {
		r.rows = map[string]domain.Vault{}
	}
	r.rows[v.ID] = v
}

func (r *indexRows) Forget(_ context.Context, vaultID string) error {
	r.forgot = append(r.forgot, vaultID)
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
	add, registry := adding(t)
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

func TestForgetTakesTheVaultOffTheListAndOutOfTheIndex(t *testing.T) {
	t.Parallel()
	kept, gone, registry := twoVaults(t)
	index := &indexRows{}

	if err := (usecase.Forget{Registry: registry, Index: index}).Execute(t.Context(), gone); err != nil {
		t.Fatal(err)
	}

	known, err := registry.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 1 || known[0].ID != kept.ID {
		t.Errorf("the list holds %v, want only %s", known, kept.Name)
	}
	if len(index.forgot) != 1 || index.forgot[0] != gone.ID {
		t.Errorf("the index was told to forget %v, want %s", index.forgot, gone.ID)
	}
}

func TestForgetLeavesTheFolderWhereItIs(t *testing.T) {
	t.Parallel()
	_, gone, registry := twoVaults(t)

	if err := (usecase.Forget{Registry: registry, Index: &indexRows{}}).Execute(t.Context(), gone); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(gone.Path, ".numen", "config.json")); err != nil {
		t.Errorf("the folder no longer carries its identity: %v", err)
	}
}

func TestForgetRefusesTheOnlyVault(t *testing.T) {
	t.Parallel()
	add, registry := adding(t)
	only, err := add.Execute(folder(t, "personal"), "")
	if err != nil {
		t.Fatal(err)
	}
	index := &indexRows{}

	err = (usecase.Forget{Registry: registry, Index: index}).Execute(t.Context(), only)
	if !errors.Is(err, usecase.ErrLastVault) {
		t.Fatalf("the last vault was answered %v", err)
	}
	known, err := registry.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 1 {
		t.Errorf("the list holds %v, want the vault still on it", known)
	}
	if len(index.forgot) != 0 {
		t.Errorf("the index was told to forget %v", index.forgot)
	}
}

func TestAVaultTheIndexCouldNotForgetStaysOnTheList(t *testing.T) {
	t.Parallel()
	_, gone, registry := twoVaults(t)
	index := &indexRows{fails: errors.New("the index is locked")}

	if err := (usecase.Forget{Registry: registry, Index: index}).Execute(t.Context(), gone); err == nil {
		t.Fatal("an index that refused was reported as success")
	}
	if _, found, err := registry.Find(gone.ID); err != nil || !found {
		t.Errorf("the vault left the list with its rows still in the index: %v %v", found, err)
	}
}
