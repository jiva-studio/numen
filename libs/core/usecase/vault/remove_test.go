package vault_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
)

func newRemove(registry *appstate.VaultRegistry) (vaults.Remove, *bin, *indexRows) {
	erase, trash, index, _ := newErase(registry)
	return vaults.NewRemove(
		vaults.NewFind(registry),
		vaults.NewUnregister(registry, index),
		erase,
	), trash, index
}

func TestRemoveConsumesAVaultOffTheListAndPreservesItsFolder(t *testing.T) {
	t.Parallel()
	retained, removed, registry := twoVaults(t)
	remove, trash, _ := newRemove(registry)

	if err := remove.Execute(t.Context(), string(removed.ID), retained.ID, false); err != nil {
		t.Fatal(err)
	}

	if _, found, err := registry.FindVault(string(removed.ID)); err != nil || found {
		t.Errorf("the vault is still on the list: %v %v", found, err)
	}
	if len(trash.moved) != 0 {
		t.Errorf("the folder went to the trash: %v", trash.moved)
	}
}

func TestRemoveTrashesTheFolderWhenAskedTo(t *testing.T) {
	t.Parallel()
	retained, removed, registry := twoVaults(t)
	remove, trash, _ := newRemove(registry)

	if err := remove.Execute(t.Context(), string(removed.ID), retained.ID, true); err != nil {
		t.Fatal(err)
	}

	if len(trash.moved) != 1 || trash.moved[0] != removed.Path {
		t.Errorf("trashed %v, want %s", trash.moved, removed.Path)
	}
}

func TestRemoveKeepsTheVaultInFrontOfThePerson(t *testing.T) {
	t.Parallel()
	_, showing, registry := twoVaults(t)
	remove, trash, index := newRemove(registry)

	for _, trashed := range []bool{false, true} {
		err := remove.Execute(t.Context(), string(showing.ID), showing.ID, trashed)
		if !errors.Is(err, vaults.ErrShowing) {
			t.Fatalf("error is %v, want ErrShowing", err)
		}
	}

	if len(trash.moved) != 0 || len(index.unregistered) != 0 {
		t.Errorf("something went: %v, %v", trash.moved, index.unregistered)
	}
	_, found, findErr := registry.FindVault(string(showing.ID))
	if findErr != nil {
		t.Fatal(findErr)
	}
	if !found {
		t.Error("the vault in front of the person left the list")
	}
}

func TestRemoveRefusesAVaultThatIsNotOnTheList(t *testing.T) {
	t.Parallel()
	retained, _, registry := twoVaults(t)
	remove, _, _ := newRemove(registry)

	err := remove.Execute(t.Context(), "nothing", retained.ID, false)

	if !errors.Is(err, vaults.ErrUnknown) {
		t.Fatalf("error is %v, want ErrUnknown", err)
	}
}
