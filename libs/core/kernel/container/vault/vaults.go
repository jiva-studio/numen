// Package vault wires vaults, opening, notes, text versions, index and open vaults.
package vault

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/trash"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/problemport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	vaultusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/check"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Config wires vaults, opening, notes, text versions, index and open vaults.
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
	SyncSetting  query.SyncSetting
	CutNotes     query.CutNotes
	SetNoteCut   func(*vaultusecase.Scan, *vaultusecase.Index)
}

// Registry opens the vault registry for this installation.
func (c Config) Registry() (vaultregistry.Ports, error) {
	list, err := c.openRegistry()
	if err != nil {
		return vaultregistry.Ports{}, err
	}
	return vaultregistry.Ports{
		Vaults: list, Lookup: list, Saver: list, Remover: list, Recorder: list, Last: list,
	}, nil
}

func (c Config) openRegistry() (*appstate.VaultRegistry, error) {
	if c.RegistryPath != "" {
		return appstate.OpenAt(c.RegistryPath), nil
	}
	return appstate.Open()
}

// VaultReaders opens vaults for reading.
func (c Config) VaultReaders() vaultport.VaultReaders {
	return vaultreader.Readers{Options: c.vaultOptions()}
}

func (c Config) writers() vaultwriter.VaultWriters {
	return vaultwriter.VaultWriters{Options: c.vaultOptions(), Locks: c.VaultLocks}
}

// NoteWriters opens vaults for writing notes.
func (c Config) NoteWriters() vaultport.NoteWriters { return c.writers() }

// FolderWriters opens vaults for filling their folders.
func (c Config) FolderWriters() vaultport.FolderWriters { return c.writers() }

// FileMovers opens vaults for moving and removing files.
func (c Config) FileMovers() vaultport.FileMovers { return c.writers() }

// WriteLocks holds the write locks of the vaults.
func (c Config) WriteLocks() vaultport.VaultLocks { return c.writers() }

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
// notes settles each note a move sent somewhere else. It is the same note.Move
// a rename goes through, so a note settles one way however it traveled.
func (c Config) Vaults(registry vaultregistry.Ports, db *vaultusecase.Index, notes note.Move) vaultusecase.Scenarios {
	// Erase is Unregister and a folder that goes, so the two hold one Unregister.
	unregister := vaultusecase.NewUnregister(registry.Vaults, registry.Remover, db.Vaults())
	erase := vaultusecase.NewErase(c.VaultIdentity(), c.Trash(), unregister)
	find := vaultregistry.NewFind(registry.Lookup)
	return vaultusecase.Scenarios{
		List:        vaultregistry.NewList(registry.Vaults),
		RecordOpen:  vaultusecase.NewRecordOpen(registry.Recorder),
		Find:        find,
		FolderCheck: check.NewFolderCheck(c.VaultReaders()),
		Remove:      vaultusecase.NewRemove(find, unregister, erase),
		Add:         vaultusecase.NewAdd(c.VaultIdentity(), registry.Vaults, registry.Lookup, registry.Saver, c.Clock),
		Rename:      vaultusecase.NewRename(registry.Vaults, registry.Saver, db.Vaults()),
		Unregister:  unregister,
		Erase:       erase,
		Move: vaultusecase.NewMove(
			c.writers(), db.Links(), db.SourcesUnder(), db.Sources(), notes),
		Import: vaultusecase.NewImport(c.writers(), c.GetImportedFiles()),
	}
}

// AddVault registers a new vault in the registry with the given root path and name.
func (c Config) AddVault(registry vaultregistry.Ports, root, name string) (domain.Vault, error) {
	return vaultusecase.NewAdd(c.VaultIdentity(), registry.Vaults, registry.Lookup, registry.Saver, c.Clock).Execute(root, name)
}

// GetKnownVaults builds the use case that lists known vaults and their folder state.
func (c Config) GetKnownVaults(registry vaultregistry.Ports) *vaultregistry.GetKnownVaults {
	recognized := vaultregistry.NewGetKnownVaults(registry.Vaults, registry.Last, c.VaultReaders())
	return &recognized
}

// GetProblems holds the vault to the standard checks.
func (Config) GetProblems(problems problemport.ProblemQueries) *vaultusecase.GetProblems {
	return vaultusecase.NewStandardGetProblems(problems)
}

// GetSummary builds the vault summary query use case.
func (Config) GetSummary(summaries noteport.VaultSummaries) *vaultregistry.GetSummary {
	return vaultregistry.NewGetSummary(summaries)
}

// PickFolder builds the folder picker use case when a dialog is provided.
func (c Config) PickFolder(dialog machineport.FolderDialog) *picker.PickFolder {
	if dialog == nil {
		return nil
	}
	pick := picker.NewPickFolder(dialog)
	return &pick
}
