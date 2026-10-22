package vault_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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

// newErase wires the use case over one registry, recording the order the folder
// and the rows go in.
func newErase(registry *appstate.VaultRegistry) (vaults.Erase, *bin, *indexRows, *[]string) {
	steps := &[]string{}
	trash := &bin{steps: steps}
	index := &indexRows{steps: steps}
	return vaults.NewErase(
		filesystem.VaultIdentity{},
		trash,
		vaults.NewUnregister(registry, index),
	), trash, index, steps
}

func TestEraseTrashesTheFolderBeforeForgettingIt(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)
	erase, trash, index, steps := newErase(registry)

	res, err := erase.Execute(t.Context(), removed)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsTrashed {
		t.Error("the folder went to the trash and the erasure says it did not")
	}

	if !slices.Equal(*steps, []string{"trash", "forget"}) {
		t.Errorf("what happened, in order: %v", *steps)
	}
	if !slices.Equal(trash.moved, []string{removed.Path}) {
		t.Errorf("trashed %v, want %s", trash.moved, removed.Path)
	}
	if !slices.Equal(index.unregistered, []domain.VaultID{removed.ID}) {
		t.Errorf("forgot %v, want %s", index.unregistered, removed.ID)
	}
	if _, found, err := registry.FindVault(string(removed.ID)); err != nil || found {
		t.Errorf("the vault is still on the list: %v %v", found, err)
	}
}

func TestEraseRefusesAFolderThatNoLongerCarriesTheIdentity(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)
	erase, trash, index, _ := newErase(registry)

	// The folder at the path a registry entry names is some other folder now.
	if err := os.RemoveAll(filepath.Join(removed.Path, ".numen")); err != nil {
		t.Fatal(err)
	}

	if _, err := erase.Execute(t.Context(), removed); !errors.Is(err, vaults.ErrUnreadable) {
		t.Fatalf("a folder that is not the vault was answered %v", err)
	}
	if len(trash.moved) != 0 {
		t.Errorf("trashed %v", trash.moved)
	}
	if len(index.unregistered) != 0 {
		t.Errorf("forgot %v", index.unregistered)
	}
	if _, found, err := registry.FindVault(string(removed.ID)); err != nil || !found {
		t.Errorf("the vault left the list: %v %v", found, err)
	}
}

func TestAFolderThatIsMissingIsForgottenAndNothingIsTrashed(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)
	erase, trash, index, _ := newErase(registry)

	if err := os.RemoveAll(removed.Path); err != nil {
		t.Fatal(err)
	}

	res, err := erase.Execute(t.Context(), removed)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsTrashed {
		t.Error("nothing was there to trash and the erasure says the folder went")
	}
	if len(trash.moved) != 0 {
		t.Errorf("trashed %v, and there was nothing there", trash.moved)
	}
	if !slices.Equal(index.unregistered, []domain.VaultID{removed.ID}) {
		t.Errorf("forgot %v, want %s", index.unregistered, removed.ID)
	}
	if _, found, err := registry.FindVault(string(removed.ID)); err != nil || found {
		t.Errorf("the vault is still on the list: %v %v", found, err)
	}
}

func TestEraseRefusesTheOnlyVaultBeforeTouchingItsFolder(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	only, err := add.Execute(folder(t, "personal"), "")
	if err != nil {
		t.Fatal(err)
	}
	erase, trash, _, _ := newErase(registry)

	if _, err := erase.Execute(t.Context(), only); !errors.Is(err, vaults.ErrLastVault) {
		t.Fatalf("the last vault was answered %v", err)
	}
	if len(trash.moved) != 0 {
		t.Errorf("trashed %v", trash.moved)
	}
	if _, err := os.Stat(only.Path); err != nil {
		t.Errorf("the folder is gone: %v", err)
	}
}

func TestEraseTouchesNothingWhenTheFolderCannotBeRead(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)
	index := &indexRows{}
	trash := &bin{}
	erase := vaults.NewErase(
		refusingIdentity{checkReadable: errors.New("permission denied")},
		trash,
		vaults.NewUnregister(registry, index),
	)

	res, err := erase.Execute(t.Context(), removed)
	if !errors.Is(err, vaults.ErrUnreadable) {
		t.Fatalf("want ErrUnreadable, got %v", err)
	}
	if res.IsTrashed || len(trash.moved) != 0 || len(index.unregistered) != 0 {
		t.Errorf("trashed %v, unregistered %v, result %+v", trash.moved, index.unregistered, res)
	}
}

func TestEraseTouchesNothingWhenTheIdentityCannotBeRead(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)
	index := &indexRows{}
	trash := &bin{}
	erase := vaults.NewErase(
		refusingIdentity{vaultID: func(string) (domain.VaultID, bool, error) { return "", false, errRefused }},
		trash,
		vaults.NewUnregister(registry, index),
	)

	_, err := erase.Execute(t.Context(), removed)
	if !errors.Is(err, errRefused) || errors.Is(err, vaults.ErrUnreadable) {
		t.Fatalf("want the rejection itself back, got %v", err)
	}
	if len(trash.moved) != 0 || len(index.unregistered) != 0 {
		t.Errorf("trashed %v, unregistered %v", trash.moved, index.unregistered)
	}
}

// Both halves of what a folder must carry are checked: that it carries an
// identity, and that it is this vault's.
func TestEraseRefusesAFolderWhoseIdentityIsNotThisVaults(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		answers func(gone domain.Vault) (domain.VaultID, bool)
	}{
		{"carries none", func(gone domain.Vault) (domain.VaultID, bool) { return gone.ID, false }},
		{"carries another", func(domain.Vault) (domain.VaultID, bool) { return "01OTHER", true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, removed, registry := twoVaults(t)
			index := &indexRows{}
			trash := &bin{}
			erase := vaults.NewErase(
				refusingIdentity{vaultID: func(string) (domain.VaultID, bool, error) {
					id, does := tc.answers(removed)
					return id, does, nil
				}},
				trash,
				vaults.NewUnregister(registry, index),
			)

			if _, err := erase.Execute(t.Context(), removed); !errors.Is(err, vaults.ErrUnreadable) {
				t.Fatalf("want ErrUnreadable, got %v", err)
			}
			if len(trash.moved) != 0 || len(index.unregistered) != 0 {
				t.Errorf("trashed %v, unregistered %v", trash.moved, index.unregistered)
			}
		})
	}
}

func TestAFolderThatCannotBeTrashedStaysOnTheList(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)
	erase, trash, index, _ := newErase(registry)
	trash.fails = errRefused

	res, err := erase.Execute(t.Context(), removed)
	if !errors.Is(err, errRefused) {
		t.Fatalf("want the rejection back, got %v", err)
	}
	if res.IsTrashed || len(index.unregistered) != 0 {
		t.Errorf("result %+v, unregistered %v", res, index.unregistered)
	}
	if _, found, err := registry.FindVault(string(removed.ID)); err != nil || !found {
		t.Errorf("the vault left the list: %v %v", found, err)
	}
}

func TestEraseKeepsTheCauseWhenTheFolderCannotBeRead(t *testing.T) {
	t.Parallel()
	_, removed, registry := twoVaults(t)
	erase := vaults.NewErase(
		refusingIdentity{checkReadable: errRefused},
		&bin{},
		vaults.NewUnregister(registry, &indexRows{}),
	)

	_, err := erase.Execute(t.Context(), removed)
	if !errors.Is(err, vaults.ErrUnreadable) || !errors.Is(err, errRefused) {
		t.Fatalf("want both the sentinel and the cause, got %v", err)
	}
}
