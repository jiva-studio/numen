package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/trash"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/check"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// EnsureVault has the folder at root on the list of vaults and returns its vault. A folder that is on the list is
// returned as it is; otherwise it is added under name, and it is recorded as the vault opened last when
// isRecordedOpened is set.
func (c Config) EnsureVault(res *Resources, root, name string, isRecordedOpened bool) (domain.Vault, error) {
	ports, err := openVaultRegistry(c.RegistryPath)
	if err != nil {
		return domain.Vault{}, err
	}
	identity := createVaultIdentity(vaultconfig.Options{ServiceDir: c.ServiceDir})
	add := newAddVault(identity, GetClock(res), ports)
	ensure := vault.NewEnsureRegistered(identity, ports.Lookup, add, ports.Recorder)
	return ensure.Execute(vault.EnsureRequest{Root: root, Name: name, IsRecordedOpened: isRecordedOpened})
}

func (c Config) openIndex(res *Resources, ctx context.Context) (*vaultIndex, error) {
	timer := GetClock(res)
	db, err := openIndexAt(ctx, c.IndexPath, index.WithQueryLog(timer, openOptionalRecordLog(c.IndexPath, res)))
	if err == nil && c.Absent {
		res.absentFiles = db.AbsentFiles()
	}
	return db, err
}

// vaultWiring holds the filesystem ports and indexing callbacks of an installation.
type vaultWiring struct {
	IndexPath    string
	RegistryPath string
	Options      vaultconfig.Options
	Readers      vaultport.VaultReaders
	Writers      vaultwriter.VaultWriters
	Watcher      vaultwatch.Watcher
	Clock        timeSource
	TextVersions *textVersions
	SyncSetting  naming.SyncSetting
	CutNotes     chunking.CutNotes
	SetNoteCut   func(*indexing.Scan)
	RecordLog    index.QueryLogFunc
	AbsentFiles  vaultport.AbsentFiles
}

func openVaultAccess(paths Paths, timer timeSource, locks *vaultwriter.Locks, watches *vaultwatch.Hub, texts *textVersions) vaultWiring {
	options := vaultconfig.Options{ServiceDir: paths.ServiceDir}
	return vaultWiring{
		IndexPath:    paths.IndexPath,
		RegistryPath: paths.RegistryPath,
		Options:      options,
		Clock:        timer,
		TextVersions: texts,
		Readers:      vaultreader.Readers{Options: options},
		Writers:      vaultwriter.VaultWriters{Options: options, Locks: locks},
		Watcher:      vaultwatch.Watcher{Options: options, Hub: watches.GetShared()},
	}
}

func openVaultRegistry(path string) (vaultregistry.Ports, error) {
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

func createVaultIdentity(options vaultconfig.Options) vaultport.VaultIdentity {
	return vaultreader.Identity{Options: options}
}

func createTrash(clock machineport.Clock) vault.Trash { return trash.New(clock) }

func createVaultUseCases(c vaultWiring, registry vaultregistry.Ports, db *vaultIndex) vaultUseCases {
	// Erase is Unregister and a folder that goes, so the two hold one Unregister.
	unregister := vault.NewUnregister(registry.Vaults, registry.Remover, db.Vaults())
	identity := createVaultIdentity(c.Options)
	erase := vault.NewErase(identity, createTrash(c.Clock), unregister)
	return vaultUseCases{
		Change: vaultChanging{
			Add:        newAddVault(identity, c.Clock, registry),
			Rename:     vault.NewRename(registry.Vaults, registry.Saver, db.Vaults()),
			Remove:     vault.NewRemove(registry.Lookup, unregister, erase),
			Unregister: unregister,
			Erase:      erase,
		},
		Tree: vault.Tree{

			Import: vault.NewImport(c.Writers, filesystem.ImportedFiles{}),
		},
	}
}

func newAddVault(identity vaultport.VaultIdentity, clock machineport.Clock, registry vaultregistry.Ports) vault.Add {
	return vault.NewAdd(identity, registry.Vaults, registry.Lookup, registry.Saver, clock)
}

func getProblems(problems check.ProblemQueries) *vault.GetProblems {
	if problems == nil {
		return vault.NewGetProblems(check.Checks{})
	}
	return vault.NewGetProblems(check.CreateStandard(problems))
}
