package container

import (
	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/serialqueue"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/problemport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
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
		Add:         vault.NewAdd(c.VaultIdentity(), registry, c.Clock()),
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

func (c Config) CheckRoot() *vault.CheckRoot {
	return vault.NewCheckRoot(c.VaultIdentity(), filesystem.HomeFolder{})
}

func (c Config) GetKnownVaults(registry vaultport.VaultRegistry) *vault.GetKnownVaults {
	recognized := vault.NewGetKnownVaults(registry, c.VaultReaders())
	return &recognized
}

// GetProblems holds the vault to the standard checks.
func (Config) GetProblems(problems problemport.ProblemQueries) *vault.GetProblems {
	return vault.NewStandardGetProblems(problems)
}

func (Config) GetSummary(queries noteport.NoteQueries) *vault.GetSummary {
	return vault.NewGetSummary(queries)
}

func (c Config) PickFolder(dialog machineport.FolderDialog) *vault.PickFolder {
	if dialog == nil {
		return nil
	}
	pick := vault.NewPickFolder(dialog)
	return &pick
}

// Resources is what every part a container assembles shares: the write lock of each vault, the
// folder watches and the access to the settings file. An application builds one with
// NewResources and names it in its Config.
type Resources struct {
	vaultLocks    *vaultwriter.Locks
	watches       *vaultwatch.Hub
	settingsFiles *settings.Files
	pluginUpdates *serialqueue.Queue
}

// NewResources returns the resources of one container.
func NewResources() *Resources {
	return &Resources{
		vaultLocks:    vaultwriter.NewLocks(),
		watches:       vaultwatch.NewHub(),
		settingsFiles: settings.NewFiles(),
		pluginUpdates: serialqueue.New(),
	}
}

// NewConfig returns c with the resources of one container. A Config built without NewConfig and
// without Resources opens writers, watches and settings that refuse to work.
func NewConfig(c Config) Config {
	if c.Resources == nil {
		c.Resources = NewResources()
	}
	return c
}

// getResources returns the resources the Config names, or empty ones whose locks, watches and
// settings access return an error.
func (c Config) getResources() *Resources {
	if c.Resources != nil {
		return c.Resources
	}
	return &Resources{vaultLocks: nil, watches: nil, settingsFiles: nil, pluginUpdates: nil}
}
