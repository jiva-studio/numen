package vault_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

func TestTheListReportsWhichVaultFoldersAreMissing(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	harbor, err := add.Execute(folder(t, "harbor"), "")
	if err != nil {
		t.Fatal(err)
	}
	orchardRoot := folder(t, "orchard")
	if _, err := add.Execute(orchardRoot, ""); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(orchardRoot); err != nil {
		t.Fatal(err)
	}

	got, err := vaults.NewListVaults(registry, vaultreader.Readers{}).Execute()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != harbor.ID || got[0].IsMissing || !got[1].IsMissing || got[1].Name != "orchard" {
		t.Errorf("listed %+v, want only the orchard marked missing", got)
	}
}

func TestAddVault_NewFolder_ReturnsTheVaultWithTheStateOfItsFolder(t *testing.T) {
	t.Parallel()
	add, _ := newAdd(t)
	root := folder(t, "shelf")
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	got, err := vaults.NewAddVault(add, vaultreader.Readers{}).Execute(root, "new")
	if err != nil || got.Path != resolved || got.Name != "new" || got.IsMissing {
		t.Errorf("added %+v, %v", got, err)
	}
	if _, err := vaults.NewAddVault(add, vaultreader.Readers{}).Execute("relative", "new"); err == nil {
		t.Error("a folder with no absolute path was added")
	}
}

func TestARenameFindsTheVaultThenNamesIt(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	harbor, err := add.Execute(folder(t, "harbor"), "")
	if err != nil {
		t.Fatal(err)
	}
	rename := vaults.NewRenameVault(registry, vaults.NewRename(registry, registry, &indexRows{}), vaultreader.Readers{})

	got, err := rename.Execute(t.Context(), string(harbor.ID), "port")
	if err != nil || got.Name != "port" || got.Path != harbor.Path {
		t.Errorf("renamed %+v, %v", got, err)
	}
	if _, err := rename.Execute(t.Context(), "nobody", "port"); !errors.Is(err, vaultdomain.ErrUnknown) {
		t.Errorf("an unknown vault answered %v", err)
	}
}
