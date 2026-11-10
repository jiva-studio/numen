package container

import (
	"context"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/trash"
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
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/check"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// Registry opens the vault registry for this installation.
func (c Config) Registry() (vaultregistry.Ports, error) {
	return vaultRegistry(c.vault())
}

// VaultReaders opens vaults for reading.
func (c Config) VaultReaders() vaultport.VaultReaders {
	return vaultReaders(c.vault())
}

// NoteWriters opens vaults for writing notes.
func (c Config) NoteWriters() vaultport.NoteWriters {
	return noteWriters(c.vault())
}

func (c Config) fileMovers() vaultport.FileMovers {
	return fileMovers(c.vault())
}

// VaultLocks holds the write locks of the vaults.
func (c Config) VaultLocks() vaultport.VaultLocks {
	return writeLocks(c.vault())
}

func (c Config) vaultWatcher() vaultport.VaultWatcher {
	return vaultWatcher(c.vault())
}

// VaultIdentity provides vault identification.
func (c Config) VaultIdentity() vaultport.VaultIdentity {
	return vaultIdentity(c.vault())
}

// Trash provides trash and recycling operations.
func (c Config) Trash() machineport.Trash {
	return vaultTrash(c.vault())
}

func (c Config) indexPath() (string, error) {
	return resolveIndexPath(c.vault())
}

// Vaults builds the list of vaults an installation holds, and the vault's own
// tree, against this installation's list and index.
func (c Config) Vaults(ports vaultregistry.Ports, db *vault.Index, notes note.Move) vault.Scenarios {
	return vaultVaults(c.vault(), ports, db, notes)
}

// AddVault registers a new vault in the registry with the given root path and name.
func (c Config) AddVault(ports vaultregistry.Ports, root, name string) (domain.Vault, error) {
	return addVault(c.vault(), ports, root, name)
}

// GetProblems holds the vault to the standard checks.
func (c Config) GetProblems(problems problemport.ProblemQueries) *vault.GetProblems {
	return getProblems(c.vault(), problems)
}

// GetSummary builds the vault summary query use case.
func (c Config) GetSummary(summaries noteport.VaultSummaries) *vaultregistry.GetSummary {
	return getSummary(c.vault(), summaries)
}

// OpenIndex opens the cache. The caller closes it.
func (c Config) OpenIndex(ctx context.Context) (*vault.Index, error) {
	return openIndex(c.vault(), ctx)
}

// UpdateIndex updates the notes at the paths given up to date in the index.
func (c Config) UpdateIndex(db *vault.Index) edit.IndexUpdater {
	return updateIndex(c.vault(), db)
}

// Moves reports each time the index file changes.
func (c Config) Moves(ctx context.Context) (<-chan struct{}, error) {
	return vaultMoves(c.vault(), ctx)
}

// Notes builds everything that acts on the notes of a vault, against this
// installation's vault readers and writers.
func (c Config) Notes(db *vault.Index, index edit.IndexUpdater) note.Scenarios {
	return vaultNotes(c.vault(), db, index)
}

// FocusNote builds the use case that shows a note in the window.
func (c Config) FocusNote(refs noteport.NoteRefs, view windowport.Window) *query.FocusNote {
	return vaultFocusNote(c.vault(), refs, view)
}

// GetNamedPaths builds the note named paths query use case.
func (c Config) GetNamedPaths(names noteport.NamedPaths) *query.GetNamedPaths {
	return getNamedPaths(c.vault(), names)
}

// Read builds the vault file reader use case.
func (c Config) Read() file.Read {
	return vaultRead(c.vault())
}

// VaultOpener is how this installation opens a vault, the way the settings say
// one is read and watched.
func (c Config) VaultOpener(db *vault.Index) *vault.VaultOpener {
	return vaultOpener(c.vault(), db)
}

func (c Config) vaultOpenerWith(
	db *vault.Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) *vault.VaultOpener {
	return vaultOpenerWith(c.vault(), db, walking, watcher)
}

// GetRefresh brings named notes up to date in the index, over the notes of the cache.
func (c Config) GetRefresh(db *vault.Index) vault.Refresh {
	return makeRefresh(c, db, db.Notes())
}

func makeRefresh(c Config, db *vault.Index, notes noteport.NoteRepository) vault.Refresh {
	return vaultMakeRefresh(c.vault(), db, notes)
}

// NewTextVersions is the text versions of the vaults' files.
func (c Config) NewTextVersions(db *vault.Index) (*textversion.Versions, error) {
	return newTextVersions(c.vault(), db)
}

// SetPluginTexts provides the text versions of the vaults to the plugins through
// the host.
func (c Config) SetPluginTexts() {
	setPluginTexts(c.vault())
}

// vaultWiring wires vaults, opening, notes, text versions, index and open vaults.
type vaultWiring struct {
	IndexPath    string
	RegistryPath string
	ServiceDir   string
	SafeMode     bool
	Clock        machineport.Clock
	PluginHost   *PluginHost
	TextVersions *textversion.Versions
	TextReaders  textdomain.Readers
	VaultLocks   *vaultwriter.Locks
	Watches      *vaultwatch.Hub
	ErrorHandler func(error)
	SyncSetting  query.SyncSetting
	CutNotes     chunking.CutNotes
	SetNoteCut   func(*vault.Scan, *vault.Index)
}

func vaultRegistry(c vaultWiring) (vaultregistry.Ports, error) {
	list, err := openRegistry(c)
	if err != nil {
		return vaultregistry.Ports{}, err
	}
	return vaultregistry.Ports{
		Vaults: list, Lookup: list, Saver: list, Remover: list, Recorder: list, Last: list,
	}, nil
}

func openRegistry(c vaultWiring) (*appstate.VaultRegistry, error) {
	if c.RegistryPath != "" {
		return appstate.OpenAt(c.RegistryPath), nil
	}
	return appstate.Open()
}

func vaultReaders(c vaultWiring) vaultport.VaultReaders {
	return vaultreader.Readers{Options: vaultOptions(c)}
}

func vaultWriters(c vaultWiring) vaultwriter.VaultWriters {
	return vaultwriter.VaultWriters{Options: vaultOptions(c), Locks: c.VaultLocks}
}

func noteWriters(c vaultWiring) vaultport.NoteWriters { return vaultWriters(c) }

func folderWriters(c vaultWiring) vaultport.FolderWriters { return vaultWriters(c) }

func fileMovers(c vaultWiring) vaultport.FileMovers { return vaultWriters(c) }

func writeLocks(c vaultWiring) vaultport.VaultLocks { return vaultWriters(c) }

func getImportedFiles(_ vaultWiring) sourceport.ImportedFiles {
	return filesystem.ImportedFiles{}
}

func vaultWatcher(c vaultWiring) vaultport.VaultWatcher {
	return vaultwatch.Watcher{Options: vaultOptions(c), Hub: c.Watches.GetShared()}
}

func vaultIdentity(c vaultWiring) vaultport.VaultIdentity {
	return vaultreader.Identity{Options: vaultOptions(c)}
}

func vaultTrash(c vaultWiring) machineport.Trash {
	return trash.New(c.Clock)
}

func vaultOptions(c vaultWiring) vaultconfig.Options {
	return vaultconfig.Options{
		ServiceDir: c.ServiceDir,
	}
}

func vaultHandleError(c vaultWiring, err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}

func vaultVaults(c vaultWiring, registry vaultregistry.Ports, db *vault.Index, notes note.Move) vault.Scenarios {
	// Erase is Unregister and a folder that goes, so the two hold one Unregister.
	unregister := vault.NewUnregister(registry.Vaults, registry.Remover, db.Vaults())
	erase := vault.NewErase(vaultIdentity(c), vaultTrash(c), unregister)
	find := vaultregistry.NewFind(registry.Lookup)
	return vault.Scenarios{
		Registry: vault.Listing{
			List:        vaultregistry.NewList(registry.Vaults),
			Find:        find,
			FolderCheck: check.NewFolderCheck(vaultReaders(c)),
		},
		Change: vault.Changing{
			Add:        vault.NewAdd(vaultIdentity(c), registry.Vaults, registry.Lookup, registry.Saver, c.Clock),
			Rename:     vault.NewRename(registry.Vaults, registry.Saver, db.Vaults()),
			Remove:     vault.NewRemove(find, unregister, erase),
			Unregister: unregister,
			Erase:      erase,
		},
		Tree: vault.Tree{
			Move:   vault.NewMove(vaultWriters(c), db.Links(), db.SourcesUnder(), db.Sources(), notes),
			Import: vault.NewImport(vaultWriters(c), getImportedFiles(c)),
		},
		RecordOpen: vault.NewRecordOpen(registry.Recorder),
	}
}

func addVault(c vaultWiring, registry vaultregistry.Ports, root, name string) (domain.Vault, error) {
	return vault.NewAdd(vaultIdentity(c), registry.Vaults, registry.Lookup, registry.Saver, c.Clock).Execute(root, name)
}

func getProblems(_ vaultWiring, problems problemport.ProblemQueries) *vault.GetProblems {
	return vault.NewStandardGetProblems(problems)
}

func getSummary(_ vaultWiring, summaries noteport.VaultSummaries) *vaultregistry.GetSummary {
	return vaultregistry.NewGetSummary(summaries)
}
