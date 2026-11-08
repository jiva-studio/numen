// Package vault wires vaults, opening, notes, text versions, index, open vaults and playback.
package vault

import (
	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/trash"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/problemport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	vaultusecase "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Vaults is what everything that acts on the vault list and vaults is called.
type Vaults = vaultusecase.Scenarios

// Config wires vaults, opening, notes, text versions, index, open vaults and playback.
type Config struct {
	IndexPath    string
	RegistryPath string
	ServiceDir   string
	SafeMode     bool
	Clock        machineport.Clock
	PluginHost   pluginport.PluginManager
	TextVersions *textversion.Versions
	TextReaders  textport.TextReaders
	VaultLocks   *vaultwriter.Locks
	Watches      *vaultwatch.Hub
	ErrorHandler func(error)
	SyncSetting  note.SyncSetting
	CutNotes     note.CutNotes
	SetNoteCut   func(*vaultusecase.Scan, *Index)
}

// Registry opens the vault registry for this installation.
func (c Config) Registry() (vaultport.VaultRegistry, error) {
	if c.RegistryPath != "" {
		return appstate.OpenAt(c.RegistryPath), nil
	}
	return appstate.Open()
}

// VaultReaders opens vaults for reading.
func (c Config) VaultReaders() vaultport.VaultReaders {
	return vaultreader.Readers{Options: c.vaultOptions()}
}

// VaultWriters opens vaults for write operations.
func (c Config) VaultWriters() vaultport.VaultWriters {
	return vaultwriter.VaultWriters{Options: c.vaultOptions(), Locks: c.VaultLocks}
}

// GetImportedFiles opens files imported from outside the vault.
func (c Config) GetImportedFiles() sourceport.ImportedFiles {
	return filesystem.ImportedFiles{}
}

// VaultWatcher monitors vaults for external filesystem changes.
func (c Config) VaultWatcher() vaultport.VaultWatcher {
	return vaultwatch.Watcher{Options: c.vaultOptions(), Hub: c.Watches}
}

// VaultIdentity provides vault identification.
func (c Config) VaultIdentity() vaultport.VaultIdentity {
	return vaultreader.Identity{Options: c.vaultOptions()}
}

// Trash provides trash and recycling operations.
func (c Config) Trash() machineport.Trash {
	return trash.New(c.Clock)
}

func (c Config) vaultOptions() vaultconfig.Options {
	return vaultconfig.Options{
		ServiceDir: c.ServiceDir,
	}
}

func (c Config) handleError(err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}

// Vaults builds the list of vaults an installation holds, and the vault's own
// tree, against this installation's list and index.
//
// notes is what settles each note a move sent somewhere else, and is the same
// note.Move a rename goes through, so a note that travelled settles one way
// however it travelled.
func (c Config) Vaults(registry vaultport.VaultRegistry, db *Index, notes note.Move) vaultusecase.Scenarios {
	// Erase is Unregister and a folder that goes, so the two hold one Unregister.
	unregister := vaultusecase.NewUnregister(registry, db.Vaults())
	erase := vaultusecase.NewErase(c.VaultIdentity(), c.Trash(), unregister)
	find := vaultusecase.NewFind(registry)
	return vaultusecase.Scenarios{
		Registry:    registry,
		List:        vaultusecase.NewList(registry),
		RecordOpen:  vaultusecase.NewRecordOpen(registry),
		Find:        find,
		FolderCheck: vaultusecase.NewFolderCheck(c.VaultReaders()),
		Remove:      vaultusecase.NewRemove(find, unregister, erase),
		Add:         vaultusecase.NewAdd(c.VaultIdentity(), registry, c.Clock),
		Rename:      vaultusecase.NewRename(registry, db.Vaults()),
		Unregister:  unregister,
		Erase:       erase,
		Move: vaultusecase.NewMove(
			c.VaultWriters(), db.Links(), db.SourcesKnown(), db.Sources(), notes),
		Import: vaultusecase.NewImport(c.VaultWriters(), c.GetImportedFiles()),
	}
}

// AddVault registers a new vault in the registry with the given root path and name.
func (c Config) AddVault(registry vaultport.VaultRegistry, root, name string) (domain.Vault, error) {
	return vaultusecase.NewAdd(c.VaultIdentity(), registry, c.Clock).Execute(root, name)
}

// GetKnownVaults builds the use case that lists known vaults and their folder state.
func (c Config) GetKnownVaults(registry vaultport.VaultRegistry) *vaultusecase.GetKnownVaults {
	recognized := vaultusecase.NewGetKnownVaults(registry, c.VaultReaders())
	return &recognized
}

// GetProblems holds the vault to the standard checks.
func (Config) GetProblems(problems problemport.ProblemQueries) *vaultusecase.GetProblems {
	return vaultusecase.NewStandardGetProblems(problems)
}

// GetSummary builds the vault summary query use case.
func (Config) GetSummary(queries noteport.NoteQueries) *vaultusecase.GetSummary {
	return vaultusecase.NewGetSummary(queries)
}

// PickFolder builds the folder picker use case when a dialog is provided.
func (c Config) PickFolder(dialog machineport.FolderDialog) *vaultusecase.PickFolder {
	if dialog == nil {
		return nil
	}
	pick := vaultusecase.NewPickFolder(dialog)
	return &pick
}
