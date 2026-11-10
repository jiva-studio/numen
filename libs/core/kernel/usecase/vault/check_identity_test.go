package vault_test

import (
	"errors"
	"os"
	"testing"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

func TestAVaultWhoseFolderCarriesItsIdentityIsReadable(t *testing.T) {
	t.Parallel()
	add, _ := newAdd(t)
	v, err := add.Execute(folder(t, "notes"), "")
	if err != nil {
		t.Fatal(err)
	}

	if err := vaults.NewCheckIdentity(vaultreader.Identity{}).Execute(v); err != nil {
		t.Fatalf("a vault in its own folder failed: %v", err)
	}
}

func TestAFolderCarryingAnotherIdentityIsNotTheVault(t *testing.T) {
	t.Parallel()
	add, _ := newAdd(t)
	v, err := add.Execute(folder(t, "notes"), "")
	if err != nil {
		t.Fatal(err)
	}
	v.ID = "another"

	if err := vaults.NewCheckIdentity(vaultreader.Identity{}).Execute(v); !errors.Is(err, vaultdomain.ErrUnreadable) {
		t.Fatalf("error is %v, want ErrUnreadable", err)
	}
}

func TestAFolderThatIsMissingIsNotReadable(t *testing.T) {
	t.Parallel()
	add, _ := newAdd(t)
	v, err := add.Execute(folder(t, "notes"), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(v.Path); err != nil {
		t.Fatal(err)
	}

	if err := vaults.NewCheckIdentity(vaultreader.Identity{}).Execute(v); !errors.Is(err, vaultdomain.ErrUnreadable) {
		t.Fatalf("error is %v, want ErrUnreadable", err)
	}
}
