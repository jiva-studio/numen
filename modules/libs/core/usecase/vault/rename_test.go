package vault_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func TestARenamedVaultIsCalledTheSameOnTheListAndInTheIndex(t *testing.T) {
	t.Parallel()
	_, renamed, registry := twoVaults(t)
	index := &indexRows{}

	got, err := (usecase.Rename{Registry: registry, Index: index}).Execute(t.Context(), renamed, "journal")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "journal" {
		t.Errorf("the vault is called %q", got.Name)
	}

	onTheList, found, err := registry.Find(renamed.ID)
	if err != nil || !found {
		t.Fatalf("the vault left the list: %v %v", found, err)
	}
	if onTheList.Name != "journal" {
		t.Errorf("the list calls it %q", onTheList.Name)
	}
	if len(index.saved) != 1 || index.saved[0].Name != "journal" {
		t.Errorf("the index was told %v", index.saved)
	}
	if _, err := os.Stat(filepath.Join(renamed.Path)); err != nil {
		t.Errorf("the folder was renamed with the vault: %v", err)
	}
}

func TestANameAnotherVaultHasIsRefused(t *testing.T) {
	t.Parallel()
	taken, renamed, registry := twoVaults(t)
	index := &indexRows{}

	// The comparison is the one the list is searched by: without case, over
	// normalised text.
	_, err := (usecase.Rename{Registry: registry, Index: index}).Execute(t.Context(), renamed, "PERSONAL")
	if !errors.Is(err, usecase.ErrNameTaken) {
		t.Fatalf("a name %s already has was answered %v", taken.Name, err)
	}
	onTheList, found, err := registry.Find(renamed.ID)
	if err != nil || !found {
		t.Fatalf("the vault left the list: %v %v", found, err)
	}
	if onTheList.Name != renamed.Name {
		t.Errorf("the list calls it %q, want %q", onTheList.Name, renamed.Name)
	}
	if len(index.saved) != 0 {
		t.Errorf("the index was told %v", index.saved)
	}
}

func TestTheNameAVaultAlreadyHasChangesNothing(t *testing.T) {
	t.Parallel()
	_, v, registry := twoVaults(t)
	index := &indexRows{}

	got, err := (usecase.Rename{Registry: registry, Index: index}).Execute(t.Context(), v, v.Name)
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

// The window holds the copy of the vault it opened, and goes on walking on it.
// The name and the path a walk carries are the copy's, and what the index says
// a vault is called stays what the list says.
func TestAWalkDoesNotPutBackTheNameAVaultHad(t *testing.T) {
	t.Parallel()
	_, showing, registry := twoVaults(t)
	index := &indexRows{}

	walk := scanner(filesystem.VaultReaders{}, openIndex(t))
	walk.Vaults = index
	if _, err := walk.Execute(t.Context(), showing); err != nil {
		t.Fatal(err)
	}

	renamed, err := (usecase.Rename{Registry: registry, Index: index}).Execute(t.Context(), showing, "journal")
	if err != nil {
		t.Fatal(err)
	}

	// A folder that went, or more changed at once than could be followed, sets
	// a walk going on the copy the window opened.
	if _, err := walk.Execute(t.Context(), showing); err != nil {
		t.Fatal(err)
	}

	if got := index.rows[showing.ID]; got.Name != renamed.Name || got.Path != renamed.Path {
		t.Errorf("the index holds %q at %s, and the list holds %q at %s",
			got.Name, got.Path, renamed.Name, renamed.Path)
	}
}
