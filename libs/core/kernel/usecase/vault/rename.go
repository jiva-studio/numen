package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Rename changes a vault's name. The name is the collection's display name and
// lives in the list this installation keeps; the folder keeps the name assigned
// by the filesystem.
type Rename struct {
	vaults vaultport.VaultList
	saver  vaultport.VaultSaver
	index  vaultport.VaultRepository
}

// RenameResult is a renamed vault and the name it had before.
type RenameResult struct {
	PreviousName string
	Vault        domain.Vault
}

// NewRename creates a Rename from the list this installation keeps and the saver of
// the list, which stores the name, and the index, which holds the row the vault's other rows point at.
func NewRename(vaults vaultport.VaultList, saver vaultport.VaultSaver, index vaultport.VaultRepository) Rename {
	return Rename{vaults: vaults, saver: saver, index: index}
}

// RenameByName renames the vault a person names.
type RenameByName struct {
	find   VaultFinding
	rename Rename
}

// NewRenameByName creates the use case that resolves the vault with find and renames it with rename.
func NewRenameByName(find VaultFinding, rename Rename) RenameByName {
	return RenameByName{find: find, rename: rename}
}

// ExecuteByName resolves the vault and renames it.
func (u RenameByName) ExecuteByName(ctx context.Context, nameOrPath, name string) (RenameResult, error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return RenameResult{}, err
	}
	renamed, err := u.rename.Execute(ctx, v, name)
	if err != nil {
		return RenameResult{}, err
	}
	return RenameResult{PreviousName: v.Name, Vault: renamed}, nil
}

// Execute answers with the vault under its new name. The name a vault already
// has is not a change and not an error.
func (u Rename) Execute(ctx context.Context, v domain.Vault, name string) (domain.Vault, error) {
	if name == "" {
		return domain.Vault{}, errors.New("the new name is empty; a vault needs a name")
	}
	if name == v.Name {
		return v, nil
	}

	recognized, err := u.vaults.List()
	if err != nil {
		return domain.Vault{}, err
	}
	if other, exists := domain.Vaults(recognized).FindByName(name, v.ID); exists {
		return domain.Vault{}, domain.NewDetailError(fmt.Sprintf("the vault at %s already has the name %s", other.Path, other.Name), vaultdomain.ErrNameTaken)
	}

	renamedVault := v
	renamedVault.Name = name
	if err := u.saver.Save(renamedVault); err != nil {
		return domain.Vault{}, err
	}
	// The index holds no name, so a rename writes nothing to it. This covers a
	// vault renamed before its first scan, which has no row yet to be scanned
	// into.
	if err := u.index.Register(ctx, renamedVault.ID); err != nil {
		return domain.Vault{}, fmt.Errorf("the index cannot record the vault %s: %w", v.Name, err)
	}
	return renamedVault, nil
}
