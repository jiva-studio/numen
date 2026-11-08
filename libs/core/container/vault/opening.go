package vault

import (
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	vaultusecase "github.com/jiva-studio/numen/libs/core/usecase/vault"
)

// VaultOpener is how this installation opens a vault.
type VaultOpener = vaultusecase.VaultOpener

// VaultChanges is what an opener reports to its callers.
type VaultChanges = vaultusecase.VaultChanges

// OpenVault is one vault an application has opened.
type OpenVault = vaultusecase.OpenVault

// BufferedNoteRepository records note writes during a vault walk.
type BufferedNoteRepository = vaultusecase.BufferedNoteRepository

// VaultOpener is how this installation opens a vault, the way the settings say
// one is read and watched.
func (c Config) VaultOpener(db *Index) *VaultOpener {
	return c.VaultOpenerWith(db, c.VaultReaders(), c.VaultWatcher())
}

// VaultOpenerWith is the same, with the watcher, and the readers the walk reads
// the vault through, in place of the installation's own.
func (c Config) VaultOpenerWith(
	db *Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) *VaultOpener {
	scan := vaultusecase.NewScan(
		walking,
		db.Vaults(),
		db.Notes(),
		db.Queries(),
		db.Maintenance(),
	)
	if c.SetNoteCut != nil {
		c.SetNoteCut(&scan, db)
	}
	scan.Walks = db.Walks()

	buffered := &BufferedNoteRepository{NoteRepository: db.Notes()}
	return vaultusecase.ConfigureVaultOpener(watcher, scan, buffered, c.MakeRefresh(db, buffered))
}

// MakeRefresh builds a vault refresh use case over db and notes.
func (c Config) MakeRefresh(db *Index, notes noteport.NoteRepository) vaultusecase.Refresh {
	refresh := vaultusecase.NewRefresh(c.VaultReaders(), db.Vaults(), notes, db.SourcesKnown(), db.Sources(), c.Clock)
	refresh.Cut = c.CutNotes
	refresh.Ledger = db.Ledger()
	return refresh
}
