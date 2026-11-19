package container

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
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
	"github.com/jiva-studio/numen/libs/core/kernel/port/problemport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/check"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func (c Config) registry() (vaultregistry.Ports, error) {
	return vaultRegistry(c.RegistryPath)
}

// AddVault registers a new vault in the registry with the given root path and name.
func (c Config) AddVault(res *Resources, root, name string) (domain.Vault, error) {
	ports, err := c.registry()
	if err != nil {
		return domain.Vault{}, err
	}
	return addVault(vaultWiring{ServiceDir: c.ServiceDir, Clock: GetClock(res)}, ports, root, name)
}

// EnsureVault has the folder at root on the list of vaults and returns its vault. A folder that is on the list is
// returned as it is; otherwise it is added under name, and it is recorded as the vault opened last when
// isRecordedOpened is set.
func (c Config) EnsureVault(res *Resources, root, name string, isRecordedOpened bool) (domain.Vault, error) {
	ports, err := c.registry()
	if err != nil {
		return domain.Vault{}, err
	}
	return ensureVault(vaultWiring{ServiceDir: c.ServiceDir, Clock: GetClock(res)}, ports, vaultregistry.EnsureRequest{Root: root, Name: name, IsRecordedOpened: isRecordedOpened})
}

// OpenIndex opens the cache. The caller closes it.
func OpenIndex(c Config, res *Resources, ctx context.Context) (*Index, error) {
	timer := GetClock(res)
	record := openRecordLog(c.IndexPath, getLogEntries(res), res.settingsFiles.GetLogLevel, timer)
	return openIndexAt(ctx, c.IndexPath, index.WithQueryLog(timer, recordLogFunc(record, res.ErrorHandler)))
}

// Queries builds everything that answers a question about the notes of a vault.
func Queries(db *Index) query.UseCases {
	return vaultQueries(db)
}

// vaultWiring wires vaults, opening, notes, text versions, index and open vaults.
type vaultWiring struct {
	IndexPath    string
	RegistryPath string
	ServiceDir   string
	Clock        machineport.Timer
	Lender       *hostendpoint.Lender
	TextPlugins  textport.TextPlugins
	TextVersions *TextVersions
	TextReaders  textdomain.Readers
	VaultLocks   *vaultwriter.Locks
	Watches      *vaultwatch.Hub
	ErrorHandler func(error)
	SyncSetting  naming.SyncSetting
	CutNotes     chunking.CutNotes
	SetNoteCut   func(*vault.Scan)
	RecordLog    index.QueryLogFunc
}

func vaultRegistry(path string) (vaultregistry.Ports, error) {
	list, err := openRegistry(path)
	if err != nil {
		return vaultregistry.Ports{}, err
	}
	return vaultregistry.Ports{
		Vaults: list, Lookup: list, Saver: list, Remover: list, Recorder: list, Last: list,
	}, nil
}

func openRegistry(path string) (*appstate.VaultRegistry, error) {
	if path != "" {
		return appstate.OpenAt(path), nil
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

func vaultVersions(c vaultWiring) vaultport.VaultVersions { return vaultWriters(c) }

func getImportedFiles(_ vaultWiring) vaultport.ImportedFiles {
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

func vaultVaults(c vaultWiring, registry vaultregistry.Ports, db *Index, notes note.Move) VaultUseCases {
	// Erase is Unregister and a folder that goes, so the two hold one Unregister.
	unregister := vault.NewUnregister(registry.Vaults, registry.Remover, db.Vaults())
	erase := vault.NewErase(vaultIdentity(c), vaultTrash(c), unregister)
	find := vaultregistry.NewFind(registry.Lookup)
	return VaultUseCases{
		Change: VaultChanging{
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
