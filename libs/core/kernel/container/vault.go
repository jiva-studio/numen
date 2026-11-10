package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/problemport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Index is the cache, exposed as the ports that the core uses.
type Index = vault.Index

// Notes is what everything that acts on the notes of a vault is called where
// it is built.
type Notes = note.Scenarios

// OpenVaults is every vault this window has open.
type OpenVaults = vault.OpenVaults[Config, *Index]

// Registry opens the vault registry for this installation.
func (c Config) Registry() (vaultport.VaultRegistry, error) {
	return c.vault().Registry()
}

// VaultReaders opens vaults for reading.
func (c Config) VaultReaders() vaultport.VaultReaders {
	return c.vault().VaultReaders()
}

// VaultWriters opens vaults for write operations.
func (c Config) VaultWriters() vaultport.VaultWriters {
	return c.vault().VaultWriters()
}

// GetImportedFiles opens files imported from outside the vault.
func (c Config) GetImportedFiles() sourceport.ImportedFiles {
	return c.vault().GetImportedFiles()
}

// VaultWatcher monitors vaults for external filesystem changes.
func (c Config) VaultWatcher() vaultport.VaultWatcher {
	return c.vault().VaultWatcher()
}

// VaultIdentity provides vault identification.
func (c Config) VaultIdentity() vaultport.VaultIdentity {
	return c.vault().VaultIdentity()
}

// Trash provides trash and recycling operations.
func (c Config) Trash() machineport.Trash {
	return c.vault().Trash()
}

func (c Config) indexPath() (string, error) {
	return c.vault().ResolveIndexPath()
}

// Vaults builds the list of vaults an installation holds, and the vault's own
// tree, against this installation's list and index.
func (c Config) Vaults(registry vaultport.VaultRegistry, db *Index, notes note.Move) vault.Scenarios {
	return c.vault().Vaults(registry, db, notes)
}

// AddVault registers a new vault in the registry with the given root path and name.
func (c Config) AddVault(registry vaultport.VaultRegistry, root, name string) (domain.Vault, error) {
	return c.vault().AddVault(registry, root, name)
}

// GetKnownVaults builds the use case that lists known vaults and their folder state.
func (c Config) GetKnownVaults(registry vaultport.VaultRegistry) *registry.GetKnownVaults {
	return c.vault().GetKnownVaults(registry)
}

// GetProblems holds the vault to the standard checks.
func (c Config) GetProblems(problems problemport.ProblemQueries) *vault.GetProblems {
	return c.vault().GetProblems(problems)
}

// GetSummary builds the vault summary query use case.
func (c Config) GetSummary(summaries noteport.VaultSummaries) *registry.GetSummary {
	return c.vault().GetSummary(summaries)
}

// PickFolder builds the folder picker use case when a dialog is provided.
func (c Config) PickFolder(dialog machineport.FolderDialog) *picker.PickFolder {
	return c.vault().PickFolder(dialog)
}

// OpenIndex opens the cache. The caller closes it.
func (c Config) OpenIndex(ctx context.Context) (*Index, error) {
	return c.vault().OpenIndex(ctx)
}

// UpdateIndex updates the notes at the paths given up to date in the index.
func (c Config) UpdateIndex(db *Index) note.IndexUpdater {
	return c.vault().UpdateIndex(db)
}

// Moves reports each time the index file changes.
func (c Config) Moves(ctx context.Context) (<-chan struct{}, error) {
	return c.vault().Moves(ctx)
}

// Notes builds everything that acts on the notes of a vault, against this
// installation's vault readers and writers.
func (c Config) Notes(db *Index, index note.IndexUpdater) note.Scenarios {
	return c.vault().Notes(db, index)
}

// FocusNote builds the use case that shows a note in the window.
func (c Config) FocusNote(refs noteport.NoteRefs, view windowport.Window) *query.FocusNote {
	return c.vault().FocusNote(refs, view)
}

// GetNamedPaths builds the note named paths query use case.
func (c Config) GetNamedPaths(names noteport.NamedPaths) *query.GetNamedPaths {
	return c.vault().GetNamedPaths(names)
}

// Read builds the vault file reader use case.
func (c Config) Read() file.Read {
	return c.vault().Read()
}

// VaultOpener is how this installation opens a vault, the way the settings say
// one is read and watched.
func (c Config) VaultOpener(db *Index) *vault.VaultOpener {
	return c.vault().VaultOpener(db)
}

// VaultOpenerWith is the same, with the watcher, and the readers the walk reads
// the vault through, in place of the installation's own.
func (c Config) VaultOpenerWith(
	db *Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) *vault.VaultOpener {
	return c.vault().VaultOpenerWith(db, walking, watcher)
}

func makeRefresh(c Config, db *Index, notes noteport.NoteRepository) vault.Refresh {
	return c.vault().MakeRefresh(db, notes)
}

// NewTextVersions is the text versions of the vaults' files.
func (c Config) NewTextVersions(db *Index) (*textversion.Versions, error) {
	return c.vault().NewTextVersions(db)
}

// SetPluginTexts provides the text versions of the vaults to the plugins through
// the host.
func (c Config) SetPluginTexts() {
	c.vault().SetPluginTexts()
}
