package file_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

const managingPlugin = "md.numen.tools.sync"

func TestCreateFolder_RefusesAVaultAnotherPluginManagesAndCreatesNothing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	managed := domain.Vault{ID: "01JTESTVAULT", Name: "Food", Path: root, ManagedBy: managingPlugin}
	create := file.NewCreateFolder(vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, nil)

	err := create.Execute(t.Context(), managed, "Dinner")

	if !errors.Is(err, vaultdomain.ErrReadOnly) {
		t.Errorf("error = %v, want ErrReadOnly", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "Dinner")); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("the folder exists after a refused create: %v", statErr)
	}
}

func TestCreateFolder_AllowsTheManagingPlugin(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	managed := domain.Vault{ID: "01JTESTVAULT", Name: "Food", Path: root, ManagedBy: managingPlugin}
	create := file.NewCreateFolder(vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, nil)

	if err := create.Execute(vaultdomain.WithCaller(t.Context(), managingPlugin), managed, "Dinner"); err != nil {
		t.Errorf("error = %v", err)
	}
}
