package vault

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	vaultusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

// VaultOpener is how this installation opens a vault, the way the settings say
// one is read and watched.
func (c Config) VaultOpener(db *vaultusecase.Index) *vaultusecase.VaultOpener {
	return c.VaultOpenerWith(db, c.VaultReaders(), c.VaultWatcher())
}

// VaultOpenerWith is the same, with the watcher, and the readers the walk reads
// the vault through, in place of the installation's own.
func (c Config) VaultOpenerWith(
	db *vaultusecase.Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) *vaultusecase.VaultOpener {
	scan := vaultusecase.NewScan(
		walking,
		db.Vaults(),
		db.Notes(),
		db.FileStats(),
		db.Maintenance(),
	)
	if c.SetNoteCut != nil {
		c.SetNoteCut(&scan, db)
	}
	scan.Walks = db.Walks()

	buffered := &vaultusecase.BufferedNoteRepository{NoteRepository: db.Notes()}
	return vaultusecase.ConfigureVaultOpener(watcher, scan, buffered, c.MakeRefresh(db, buffered))
}

// MakeRefresh builds a vault refresh use case over db and notes.
func (c Config) MakeRefresh(db *vaultusecase.Index, notes noteport.NoteRepository) vaultusecase.Refresh {
	refresh := vaultusecase.NewRefresh(c.VaultReaders(), db.Vaults(), notes, db.RemovalLookup(), db.Sources(), c.Clock)
	refresh.Cut = c.CutNotes
	refresh.Ledger = db.Ledger()
	return refresh
}
