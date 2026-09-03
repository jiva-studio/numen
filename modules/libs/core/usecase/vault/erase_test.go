package vault_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// bin is the place this machine keeps what a person deleted.
type bin struct {
	moved []string
	fails error
	steps *[]string
}

func (b *bin) Trash(path string) error {
	b.moved = append(b.moved, path)
	if b.steps != nil {
		*b.steps = append(*b.steps, "trash")
	}
	return b.fails
}

// erasing wires the use case over one registry, recording the order the folder
// and the rows go in.
func erasing(registry *appstate.VaultRegistry) (usecase.Erase, *bin, *indexRows, *[]string) {
	steps := &[]string{}
	trash := &bin{steps: steps}
	index := &indexRows{steps: steps}
	return usecase.Erase{
		Identity: filesystem.VaultIdentity{},
		Trash:    trash,
		Forget:   usecase.Forget{Registry: registry, Index: index},
	}, trash, index, steps
}

func TestEraseTrashesTheFolderBeforeForgettingIt(t *testing.T) {
	t.Parallel()
	_, gone, registry := twoVaults(t)
	erase, trash, index, steps := erasing(registry)

	if err := erase.Execute(t.Context(), gone); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(*steps, []string{"trash", "forget"}) {
		t.Errorf("what happened, in order: %v", *steps)
	}
	if !slices.Equal(trash.moved, []string{gone.Path}) {
		t.Errorf("trashed %v, want %s", trash.moved, gone.Path)
	}
	if !slices.Equal(index.forgot, []string{string(gone.ID)}) {
		t.Errorf("the index was told to forget %v, want %s", index.forgot, string(gone.ID))
	}
	if _, found, err := registry.Find(string(gone.ID)); err != nil || found {
		t.Errorf("the vault is still on the list: %v %v", found, err)
	}
}

func TestEraseRefusesAFolderThatNoLongerCarriesTheIdentity(t *testing.T) {
	t.Parallel()
	_, gone, registry := twoVaults(t)
	erase, trash, index, _ := erasing(registry)

	// The folder at the path a registry entry names is some other folder now.
	if err := os.RemoveAll(filepath.Join(gone.Path, ".numen")); err != nil {
		t.Fatal(err)
	}

	if err := erase.Execute(t.Context(), gone); !errors.Is(err, usecase.ErrUnreadable) {
		t.Fatalf("a folder that is not the vault was answered %v", err)
	}
	if len(trash.moved) != 0 {
		t.Errorf("trashed %v", trash.moved)
	}
	if len(index.forgot) != 0 {
		t.Errorf("the index was told to forget %v", index.forgot)
	}
	if _, found, err := registry.Find(string(gone.ID)); err != nil || !found {
		t.Errorf("the vault left the list: %v %v", found, err)
	}
}

func TestAFolderThatIsGoneIsForgottenAndNothingIsTrashed(t *testing.T) {
	t.Parallel()
	_, gone, registry := twoVaults(t)
	erase, trash, index, _ := erasing(registry)

	if err := os.RemoveAll(gone.Path); err != nil {
		t.Fatal(err)
	}

	if err := erase.Execute(t.Context(), gone); err != nil {
		t.Fatal(err)
	}
	if len(trash.moved) != 0 {
		t.Errorf("trashed %v, and there was nothing there", trash.moved)
	}
	if !slices.Equal(index.forgot, []string{string(gone.ID)}) {
		t.Errorf("the index was told to forget %v, want %s", index.forgot, string(gone.ID))
	}
	if _, found, err := registry.Find(string(gone.ID)); err != nil || found {
		t.Errorf("the vault is still on the list: %v %v", found, err)
	}
}

func TestEraseRefusesTheOnlyVaultBeforeTouchingItsFolder(t *testing.T) {
	t.Parallel()
	add, registry := adding(t)
	only, err := add.Execute(folder(t, "personal"), "")
	if err != nil {
		t.Fatal(err)
	}
	erase, trash, _, _ := erasing(registry)

	if err := erase.Execute(t.Context(), only); !errors.Is(err, usecase.ErrLastVault) {
		t.Fatalf("the last vault was answered %v", err)
	}
	if len(trash.moved) != 0 {
		t.Errorf("trashed %v", trash.moved)
	}
	if _, err := os.Stat(only.Path); err != nil {
		t.Errorf("the folder is gone: %v", err)
	}
}
