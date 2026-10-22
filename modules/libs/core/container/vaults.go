package container

import (
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// Vaults is what everything that acts on the vault list and vaults is called.
type Vaults = vault.Scenarios

// Vaults builds the list of vaults an installation holds, and the vault's own
// tree, against this installation's list and index.
//
// notes is what settles each note a move sent somewhere else, and is the same
// note.Move a rename goes through, so a note that travelled settles one way
// however it travelled.
func (c Config) Vaults(registry vaultport.VaultRegistry, db *Index, notes note.Move) vault.Scenarios {
	// Erase is Unregister and a folder that goes, so the two hold one Unregister.
	unregister := vault.NewUnregister(registry, db.Vaults())
	erase := vault.NewErase(c.VaultIdentity(), c.Trash(), unregister)
	find := vault.NewFind(registry)
	return vault.Scenarios{
		Registry:    registry,
		List:        vault.NewList(registry),
		RecordOpen:  vault.NewRecordOpen(registry),
		Find:        find,
		FolderCheck: vault.NewFolderCheck(c.VaultReaders()),
		Remove:      vault.NewRemove(find, unregister, erase),
		Add:         vault.NewAdd(c.VaultIdentity(), registry, time.Now),
		Rename:      vault.NewRename(registry, db.Vaults()),
		Unregister:  unregister,
		Erase:       erase,
		Move: vault.NewMove(
			c.VaultWriters(), db.Links(), db.SourcesKnown(), db.Sources(), notes),
		Import: vault.NewImport(c.VaultWriters(), c.GetImportedFiles()),
	}
}

// AddVault registers a new vault in the registry with the given root path and name.
func (c Config) AddVault(registry vaultport.VaultRegistry, root, name string) (domain.Vault, error) {
	return vault.NewAdd(c.VaultIdentity(), registry, c.Clock()).Execute(root, name)
}
