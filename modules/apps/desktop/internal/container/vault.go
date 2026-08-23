package container

import (
	"os"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

// FirstVaultName is what the first vault is called, and what its folder is
// named inside the documents folder.
const FirstVaultName = "numen"

// FirstVault is somewhere to write, made for a person who has added nothing.
//
// It makes a folder where this system keeps documents, and that folder is
// theirs: ordinary files in an ordinary place, which they may move or replace
// with one of their own.
func (c Config) FirstVault(registry port.VaultRegistry) (domain.Vault, error) {
	documents, err := filesystem.Documents()
	if err != nil {
		return domain.Vault{}, err
	}
	root := filepath.Join(documents, FirstVaultName)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return domain.Vault{}, err
	}
	// A folder that is already a vault keeps the identity it has.
	if _, err := filesystem.Initialize(root, c.VaultOptions().ServiceDir, time.Now()); err != nil {
		return domain.Vault{}, err
	}
	return usecase.Add{
		Identity: c.VaultIdentity(),
		Registry: registry,
		Now:      time.Now,
	}.Execute(root, FirstVaultName)
}
