package vault_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

func TestImportRefusesAVaultAnotherPluginManagesAndCopiesNothing(t *testing.T) {
	t.Parallel()
	v := vaulttest.NewVault(t, map[string]string{"Entropy.md": "# Entropy\n"})
	v.ManagedBy = managingPlugin
	from := createOutsideFolder(t, map[string]string{"Notes.md": "# Notes\n"})
	bring := vaults.NewImport(vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, filesystem.ImportedFiles{})

	_, err := bring.Execute(t.Context(), v, "", []string{filepath.Join(from, "Notes.md")})

	if !errors.Is(err, vaultdomain.ErrReadOnly) {
		t.Errorf("error = %v, want ErrReadOnly", err)
	}
	if _, statErr := os.Stat(filepath.Join(v.Path, "Notes.md")); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("the file arrived in a read only vault: %v", statErr)
	}
}

func TestImportAllowsThePluginThatManagesTheVault(t *testing.T) {
	t.Parallel()
	v := vaulttest.NewVault(t, map[string]string{"Entropy.md": "# Entropy\n"})
	v.ManagedBy = managingPlugin
	from := createOutsideFolder(t, map[string]string{"Notes.md": "# Notes\n"})
	bring := vaults.NewImport(vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, filesystem.ImportedFiles{})

	_, err := bring.Execute(vaultdomain.WithCaller(t.Context(), managingPlugin), v, "", []string{filepath.Join(from, "Notes.md")})

	if err != nil || readArrivedFile(t, v.Path, "Notes.md") != "# Notes\n" {
		t.Errorf("error = %v, want the file to arrive", err)
	}
}

func TestAddingAManagedFolderAgainFromTheWindowKeepsItsManager(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	root := folder(t, "Food")
	if _, err := add.ExecuteManaged(root, "Food", managingPlugin); err != nil {
		t.Fatal(err)
	}

	again, err := add.Execute(root, "Food")

	if err != nil || again.ManagedBy != managingPlugin {
		t.Errorf("added = %+v, error = %v, want the manager kept", again, err)
	}
	if listed, listErr := registry.List(); listErr != nil || len(listed) != 1 || listed[0].ManagedBy != managingPlugin {
		t.Errorf("list = %+v, error = %v, want one vault still managed", listed, listErr)
	}
}
