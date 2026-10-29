package container

import (
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
)

// VaultOpener is how this installation opens a vault.
type VaultOpener = vault.VaultOpener

// VaultChanges is what an opener reports to its callers.
type VaultChanges = vault.VaultChanges

// OpenVault is one vault an application has opened.
type OpenVault = vault.OpenVault

type bufferedNoteRepository = vault.BufferedNoteRepository

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
	scan := vault.NewScan(
		walking,
		db.Vaults(),
		db.Notes(),
		db.Queries(),
		db.Maintenance(),
	)
	scan.Cut = c.CutNotes()
	scan.Walks = db.Walks()

	buffered := &bufferedNoteRepository{NoteRepository: db.Notes()}
	return vault.ConfigureVaultOpener(watcher, scan, buffered, makeRefresh(c, db, buffered))
}

func makeRefresh(c Config, db *Index, notes noteport.NoteRepository) vault.Refresh {
	refresh := vault.NewRefresh(c.VaultReaders(), db.Vaults(), notes, db.SourcesKnown(), db.Sources(), c.Clock())
	refresh.Cut = c.CutNotes()
	refresh.Ledger = db.Ledger()
	return refresh
}
