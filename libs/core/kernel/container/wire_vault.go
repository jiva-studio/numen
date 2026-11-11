package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
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

func (c Config) registry() (vaultregistry.Ports, error) {
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

// AddVault registers a new vault in the registry with the given root path and name.
func (c Config) AddVault(root, name string) (domain.Vault, error) {
	ports, err := c.registry()
	if err != nil {
		return domain.Vault{}, err
	}
	return addVault(vaultOf(c), ports, root, name)
}

// EnsureVault has the folder at root on the list of vaults and returns its vault. A folder that is on the list is
// returned as it is; otherwise it is added under name, and it is recorded as the vault opened last when
// isRecordedOpened is set.
func (c Config) EnsureVault(root, name string, isRecordedOpened bool) (domain.Vault, error) {
	ports, err := c.registry()
	if err != nil {
		return domain.Vault{}, err
	}
	return ensureVault(vaultOf(c), ports, vaultregistry.EnsureRequest{Root: root, Name: name, IsRecordedOpened: isRecordedOpened})
}

// OpenIndex opens the cache. The caller closes it.
func OpenIndex(c Config, ctx context.Context) (*Index, error) {
	return openIndex(vaultOf(c), ctx)
}

// UpdateIndex updates the notes at the paths given up to date in the index.
func UpdateIndex(c Config, db *Index) edit.IndexUpdater {
	return updateIndex(vaultOf(c), db)
}

// Moves reports each time the index file changes.
func Moves(c Config, ctx context.Context) (<-chan struct{}, error) {
	return vaultMoves(vaultOf(c), ctx)
}

// Notes builds everything that changes the notes of a vault, against this
// installation's vault readers and writers. No view receives a move.
func Notes(c Config, db *Index, index edit.IndexUpdater) note.Scenarios {
	return vaultNotes(vaultOf(c), db, index, noteViews{Writes: note.OpenGate{}})
}

// Queries builds everything that answers a question about the notes of a vault.
func Queries(db *Index) query.Scenarios {
	return vaultQueries(db)
}

// VaultOpener is how this installation opens a vault, the way the settings say
// one is read and watched.
func VaultOpener(c Config, db *Index) *vault.VaultOpener {
	return vaultOpener(vaultOf(c), db)
}

// GetRefresh brings named notes up to date in the index, over the notes of the cache.
func GetRefresh(c Config, db *Index) vault.Refresh {
	return makeRefresh(c, db, db.Notes())
}

func makeRefresh(c Config, db *Index, notes noteport.NoteRepository) vault.Refresh {
	return vaultMakeRefresh(vaultOf(c), db, notes)
}

// vaultWiring wires vaults, opening, notes, text versions, index and open vaults.
type vaultWiring struct {
	IndexPath    string
	RegistryPath string
	ServiceDir   string
	Clock        machineport.Clock
	PluginHost   *PluginHost
	RunningHost  *PluginHost
	TextVersions *textversion.Versions
	TextReaders  textdomain.Readers
	VaultLocks   *vaultwriter.Locks
	Watches      *vaultwatch.Hub
	ErrorHandler func(error)
	SyncSetting  naming.SyncSetting
	CutNotes     chunking.CutNotes
	SetNoteCut   func(*vault.Scan)
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

func vaultVaults(c vaultWiring, registry vaultregistry.Ports, db *Index, notes note.Move) vault.Scenarios {
	// Erase is Unregister and a folder that goes, so the two hold one Unregister.
	unregister := vault.NewUnregister(registry.Vaults, registry.Remover, db.Vaults())
	erase := vault.NewErase(vaultIdentity(c), vaultTrash(c), unregister)
	find := vaultregistry.NewFind(registry.Lookup)
	return vault.Scenarios{
		Change: vault.Changing{
			Add:        newAddVault(c, registry),
			Rename:     vault.NewRename(registry.Vaults, registry.Saver, db.Vaults()),
			Remove:     vault.NewRemove(find.Execute, unregister, erase),
			Unregister: unregister,
			Erase:      erase,
		},
		Tree: vault.Tree{
			Move:   vault.NewMove(vaultWriters(c), db.NoteQueries(), db.Sources(), db.Sources(), notes),
			Import: vault.NewImport(vaultWriters(c), getImportedFiles(c)),
		},
		RecordOpen: vault.NewRecordOpen(registry.Recorder),
	}
}

func newAddVault(c vaultWiring, registry vaultregistry.Ports) vault.Add {
	return vault.NewAdd(vaultIdentity(c), registry.Vaults, registry.Lookup, registry.Saver, c.Clock)
}

func addVault(c vaultWiring, registry vaultregistry.Ports, root, name string) (domain.Vault, error) {
	return newAddVault(c, registry).Execute(root, name)
}

func ensureVault(c vaultWiring, registry vaultregistry.Ports, request vaultregistry.EnsureRequest) (domain.Vault, error) {
	return vaultregistry.NewEnsureRegistered(vaultIdentity(c), registry.Lookup, newAddVault(c, registry), registry.Recorder).Execute(request)
}

func getProblems(_ vaultWiring, problems problemport.ProblemQueries) *vault.GetProblems {
	if problems == nil {
		return vault.NewGetProblems(check.Checks{})
	}
	return vault.NewGetProblems(check.Standard(problems))
}

func vaultListing(c vaultWiring, registry vaultregistry.Ports) vaultregistry.Listing {
	return vaultregistry.Listing{
		List:        vaultregistry.NewList(registry.Vaults),
		Find:        vaultregistry.NewFind(registry.Lookup),
		FolderCheck: vaultregistry.NewFolderCheck(vaultReaders(c)),
	}
}
