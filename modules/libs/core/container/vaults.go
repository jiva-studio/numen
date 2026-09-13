package container

import (
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Vaults builds the list of vaults an installation holds, and the vault's own
// tree, against this installation's list and index.
//
// notes is what settles each note a move sent somewhere else, and is the same
// note.Move a rename goes through, so a note that travelled settles one way
// however it travelled.
func (c Config) Vaults(registry port.VaultRegistry, db *Index, notes note.Move) vault.Scenarios {
	// Erase is Forget and a folder that goes, so the two hold one Forget.
	forget := vault.NewForget(registry, db.Vaults())
	return vault.Scenarios{
		Registry: registry,
		Add:      vault.NewAdd(c.VaultIdentity(), registry, time.Now),
		Rename:   vault.NewRename(registry, db.Vaults()),
		Forget:   forget,
		Erase:    vault.NewErase(c.VaultIdentity(), c.Trash(), forget),
		Move: vault.NewMove(
			c.VaultWriters(), db.Links(), db.SourcesKnown(), db.Sources(), notes),
		Import: vault.NewImport(c.VaultWriters(), c.GetImportedFiles()),
	}
}
