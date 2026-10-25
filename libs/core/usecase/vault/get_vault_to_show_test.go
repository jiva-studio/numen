package vault_test

import (
	"errors"
	"testing"

	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestAVaultNamedIsTheOneChosen(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	first, err := add.Execute(folder(t, "first"), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := add.Execute(folder(t, "second"), ""); err != nil {
		t.Fatal(err)
	}

	got, err := vaults.NewGetVaultToShow(registry).Execute(first.Name)
	if err != nil || got.ID != first.ID {
		t.Fatalf("chose %+v, %v; want %s", got, err, first.Name)
	}
}

func TestAVaultNamedAndNotOnTheListIsRefused(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	if _, err := add.Execute(folder(t, "first"), ""); err != nil {
		t.Fatal(err)
	}

	if _, err := vaults.NewGetVaultToShow(registry).Execute("elsewhere"); !errors.Is(err, vaults.ErrUnknown) {
		t.Fatalf("error is %v, want ErrUnknown", err)
	}
}

func TestWithNothingNamedTheVaultShownLastIsChosen(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	if _, err := add.Execute(folder(t, "first"), ""); err != nil {
		t.Fatal(err)
	}
	second, err := add.Execute(folder(t, "second"), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.RecordOpened(second.ID); err != nil {
		t.Fatal(err)
	}

	got, err := vaults.NewGetVaultToShow(registry).Execute("")
	if err != nil || got.ID != second.ID {
		t.Fatalf("chose %+v, %v; want the vault opened last", got, err)
	}
}

func TestWithNothingShownYetTheFirstVaultIsChosen(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	target, err := add.Execute(folder(t, "first"), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := add.Execute(folder(t, "second"), ""); err != nil {
		t.Fatal(err)
	}

	got, err := vaults.NewGetVaultToShow(registry).Execute("")
	if err != nil || got.ID != target.ID {
		t.Fatalf("chose %+v, %v; want the first on the list", got, err)
	}
}

func TestAnInstallationContainingNoVaultChoosesNone(t *testing.T) {
	t.Parallel()
	got, err := vaults.NewGetVaultToShow(registryAt(t)).Execute("")
	if err != nil || got != (domain.Vault{}) {
		t.Fatalf("chose %+v, %v; want no vault", got, err)
	}
}
