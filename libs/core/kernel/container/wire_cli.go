package container

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/operationresult"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// cliRun is what the parts of one run of the command line share: the one list of tasks the plugin work of
// the run reports into, the lock of the models, and what Close closes.
type cliRun struct {
	tasks   *run.Tasks
	models  *run.Lock
	closers []func() error
}

// OpenCLI returns the session of one run of the command line over the installation c describes. It opens
// nothing: each command opens what it works over, and Close closes it.
func OpenCLI(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources) cli.Session {
	exec = getPluginExecution(exec)
	r := &cliRun{tasks: run.NewTasks(), models: &run.Lock{}}
	return cli.NewSession(
		func() (cli.Vaults, error) {
			_, _, v := buildVaultSearch(c, exec, prefs, endpoints, res, nil)
			return openCLIVaults(v)
		},
		func(ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
			config := c
			config.ShouldRebuildIndex = options.Rebuild
			install := newPluginInstall(config, endpoints, res)
			settingsDeps, searchDeps, vaultDeps := buildVaultSearch(config, exec, prefs, endpoints, res, nil)
			ports, err := vaultRegistry(vaultDeps.RegistryPath)
			if err != nil {
				return cli.Indexed{}, err
			}
			db, err := openIndex(vaultDeps, ctx)
			if err != nil {
				return cli.Indexed{}, err
			}
			r.closers = append(r.closers, db.Close)
			index := cliIndex{run: r, db: db, ports: ports, report: options.ErrorHandler, openWork: newCLIPluginWork(install, exec, r)}
			return newCLIIndexed(index, install, exec, settingsDeps, searchDeps, vaultDeps), nil
		},
		func() cli.PluginCommands {
			install := newPluginInstall(c, endpoints, res)
			settingsDeps, searchDeps, vaultDeps := buildVaultSearch(c, exec, prefs, endpoints, res, nil)
			commands := newPluginCommands(settingsDeps, vaultDeps, exec, nil)
			return getCLIPluginCommands(c, commands, newCLIPluginWork(install, exec, r), searchDeps, vaultDeps)
		},
		func() ([]string, error) { return closeCLI(r) },
	)
}

func openCLIVaults(vaultDeps vaultWiring) (cli.Vaults, error) {
	ports, err := vaultRegistry(vaultDeps.RegistryPath)
	if err != nil {
		return cli.Vaults{}, err
	}
	find := registry.NewFind(ports.Lookup).Execute
	return cli.Vaults{
		Add:        vault.NewAdd(vaultIdentity(vaultDeps), ports.Vaults, ports.Lookup, ports.Saver, vaultDeps.Clock),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, vaultReaders(vaultDeps)),
		RecordOpen: vault.NewRecordOpenByName(find, vault.NewRecordOpen(ports.Recorder)),
	}, nil
}

type cliPluginWork func(vaultWiring, *Index, domain.Vault) (*run.PluginWork, error)

func newCLIPluginWork(install pluginInstall, exec *PluginExecution, r *cliRun) cliPluginWork {
	return func(vaultDeps vaultWiring, db *Index, v domain.Vault) (*run.PluginWork, error) {
		exec.Runtime.PluginBinding.SetVaultPath(v.Path)
		if err := installCurrentVault(install, vaultDeps, db, v); err != nil {
			return nil, err
		}
		work := pluginsOpenPluginWork(install, exec, r.tasks, r.models)
		setPluginTexts(vaultDeps)
		return work, nil

	}
}

type cliIndex struct {
	ports    registry.Ports
	openWork cliPluginWork
	run      *cliRun
	db       *Index
	report   window.ErrorHandler
}

func newCLIIndexed(index cliIndex, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, searchDeps searchWiring, vaultDeps vaultWiring) cli.Indexed {
	db, ports := index.db, index.ports
	find := registry.NewFind(ports.Lookup).Execute
	unregister := vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults())
	prepareScan := func(ctx context.Context, v domain.Vault) (indexing.ReadWholeVault, error) {
		ranking, storage := openIndexedVault(searchDeps, vaultDeps, db)
		return prepareCLIScan(index, ctx, v, ranking, storage)
	}
	prepareSearch := func(ctx context.Context, v domain.Vault) (searchusecase.Search, error) {
		ranking, storage := openIndexedVault(searchDeps, vaultDeps, db)
		return prepareCLISearch(index, ctx, v, ranking, storage)
	}
	prepareQuery := func(ctx context.Context, v domain.Vault) (*searchqueryusecase.FindSearchResults, error) {
		ranking, storage := openIndexedVault(searchDeps, vaultDeps, db)
		_, err := index.openWork(storage, db, v)
		if err != nil {
			return nil, err
		}
		r := index.run
		errorHandler := index.report
		embedder, why := openCLIEmbedder(r, ranking, ctx)
		reportSearchFallback(errorHandler, why)
		setModelCalls(install, exec, settingsDeps, storage, run.NewConnectionNoticeTasks(r.tasks).Execute)
		return newFindSearchResults(openRerankDeps(install, exec, settingsDeps, storage), settingsDeps, ranking, db, embedder, errorHandler), nil
	}
	return cli.Indexed{
		Changes: cli.VaultChanges{
			Rename:     vault.NewRenameByName(find, vault.NewRename(ports.Vaults, ports.Saver, db.Vaults())),
			Unregister: vault.NewUnregisterByName(find, unregister),
			Erase:      vault.NewEraseByName(find, vault.NewErase(vaultIdentity(vaultDeps), vaultTrash(vaultDeps), unregister)),
		},
		Scan: cli.Scan{
			Scan: indexing.NewScanVault(indexing.NewReadWholeVaultByName(find, prepareScan), db.NoteQueries()),
			Now:  vaultDeps.Clock.Now,
		},
		Search:           cli.Search{Search: searchusecase.NewByName(find, prepareSearch)},
		Links:            query.NewShowLinksByName(find, Queries(db).Refer.Links),
		Problems:         vault.NewGetProblemsByName(find, getProblems(vaultDeps, db.NoteProblems())),
		Query:            searchqueryusecase.NewFindSearchResultsByName(find, prepareQuery),
		OperationResults: operationresult.NewClearOperationResults(db.OperationResults(operationResultRows)),
	}
}

func prepareCLIScan(i cliIndex, ctx context.Context, v domain.Vault, searchDeps searchWiring, vaultDeps vaultWiring) (indexing.ReadWholeVault, error) {
	db, r := i.db, i.run
	_, err := i.openWork(vaultDeps, db, v)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	embedder, why := openCLIEmbedder(r, searchDeps, ctx)
	read, err := readWholeVault(searchDeps, ctx, db, embedder, v, vaultMakeRefresh(vaultDeps, db, db.Notes()))
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	read.Unembedded = why
	return read, nil
}

func prepareCLISearch(i cliIndex, ctx context.Context, v domain.Vault, searchDeps searchWiring, vaultDeps vaultWiring) (searchusecase.Search, error) {
	db, r := i.db, i.run
	errorHandler := i.report
	_, err := i.openWork(vaultDeps, db, v)
	if err != nil {
		return searchusecase.Search{}, err
	}
	embedder, why := openCLIEmbedder(r, searchDeps, ctx)
	reportSearchFallback(errorHandler, why)
	return newSearch(searchDeps, db, embedder, errorHandler, nil), nil
}

func reportSearchFallback(errorHandler window.ErrorHandler, err error) {
	if err != nil && errorHandler != nil {
		errorHandler(err)
	}
}

func openCLIEmbedder(r *cliRun, searchDeps searchWiring, ctx context.Context) (searchsourceport.Source, error) {
	embedder, closeEmbedder, why := searchSource(searchDeps, ctx)
	if closeEmbedder != nil {
		r.closers = append(r.closers, closeEmbedder)
	}
	return embedder, why
}

func getCLIPluginCommands(c Config, commands PluginCommandUseCases, openWork cliPluginWork, searchDeps searchWiring, vaultDeps vaultWiring) cli.PluginCommands {
	find := func(nameOrPath string) (domain.Vault, error) { return findCLIVault(c, nameOrPath) }
	return cli.PluginCommands{
		List:   commands.List,
		ByName: plugincommand.NewRunByName(find),
		Open: func(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
			closeWork, err := openCLIPluginWork(openWork, ctx, v, searchDeps, vaultDeps)
			if err != nil {
				return plugincommand.Run{}, nil, err
			}
			return commands.Run, closeWork, nil
		},
	}
}

func findCLIVault(c Config, nameOrPath string) (domain.Vault, error) {
	ports, err := c.registry()
	if err != nil {
		return domain.Vault{}, err
	}
	return registry.NewFind(ports.Lookup).Execute(nameOrPath)
}

func openCLIPluginWork(openWork cliPluginWork, ctx context.Context, v domain.Vault, searchDeps searchWiring, vaultDeps vaultWiring) (func() error, error) {
	db, err := openIndex(vaultDeps, ctx)
	if err != nil {
		return nil, err
	}
	_, storage := openIndexedVault(searchDeps, vaultDeps, db)
	pluginWork, err := openWork(storage, db, v)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return run.CloseAfterWork(ctx, pluginWork, db.Close), nil
}

func closeCLI(r *cliRun) ([]string, error) {
	var err error
	for _, closeOne := range slices.Backward(r.closers) {
		err = errors.Join(err, closeOne())
	}
	return cli.FormatTaskFailures(r.tasks.List()), err
}
