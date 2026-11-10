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
	return vaultRegistry(vaultOf(c))
}

// VaultReaders opens vaults for reading.
func VaultReaders(c Config) vaultport.VaultReaders {
	return vaultReaders(vaultOf(c))
}

// NoteWriters opens vaults for writing notes.
func NoteWriters(c Config) vaultport.NoteWriters {
	return noteWriters(vaultOf(c))
}

// VaultLocks holds the write locks of the vaults.
func VaultLocks(c Config) vaultport.VaultLocks {
	return writeLocks(vaultOf(c))
}

// VaultIdentity provides vault identification.
func VaultIdentity(c Config) vaultport.VaultIdentity {
	return vaultIdentity(vaultOf(c))
}

// AddVault registers a new vault in the registry with the given root path and name.
func (c Config) AddVault(ports vaultregistry.Ports, root, name string) (domain.Vault, error) {
	return addVault(vaultOf(c), ports, root, name)
}

// OpenIndex opens the cache. The caller closes it.
func OpenIndex(c Config, ctx context.Context) (*vault.Index, error) {
	return openIndex(vaultOf(c), ctx)
}

// UpdateIndex updates the notes at the paths given up to date in the index.
func UpdateIndex(c Config, db *vault.Index) edit.IndexUpdater {
	return updateIndex(vaultOf(c), db)
}

// Moves reports each time the index file changes.
func Moves(c Config, ctx context.Context) (<-chan struct{}, error) {
	return vaultMoves(vaultOf(c), ctx)
}

// Notes builds everything that acts on the notes of a vault, against this
// installation's vault readers and writers.
func Notes(c Config, db *vault.Index, index edit.IndexUpdater) note.Scenarios {
	return vaultNotes(vaultOf(c), db, index)
}

// VaultOpener is how this installation opens a vault, the way the settings say
// one is read and watched.
func VaultOpener(c Config, db *vault.Index) *vault.VaultOpener {
	return vaultOpener(vaultOf(c), db)
}

// GetRefresh brings named notes up to date in the index, over the notes of the cache.
func GetRefresh(c Config, db *vault.Index) vault.Refresh {
	return makeRefresh(c, db, db.Stores.Notes)
}

func makeRefresh(c Config, db *vault.Index, notes noteport.NoteRepository) vault.Refresh {
	return vaultMakeRefresh(vaultOf(c), db, notes)
}

// NewTextVersions is the text versions of the vaults' files.
func NewTextVersions(c Config, db *vault.Index) (*textversion.Versions, error) {
	return newTextVersions(vaultOf(c), db)
}

// SetPluginTexts provides the text versions of the vaults to the plugins through
// the host.
func SetPluginTexts(c Config) {
	setPluginTexts(vaultOf(c))
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
	unregister := vault.NewUnregister(registry.Vaults, registry.Remover, db.Stores.Vaults)
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
			Rename:     vault.NewRename(registry.Vaults, registry.Saver, db.Stores.Vaults),
			Remove:     vault.NewRemove(find, unregister, erase),
			Unregister: unregister,
			Erase:      erase,
		},
		Tree: vault.Tree{
			Move:   vault.NewMove(vaultWriters(c), db.NoteQueries.Links, db.Source.SourcesUnder, db.Source.Sources, notes),
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
