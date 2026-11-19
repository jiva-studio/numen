package container

import (
	"context"
	"errors"
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
	r := &cliRun{tasks: run.NewTasks(), models: &run.Lock{}}
	return cli.NewSession(
		func() (cli.Vaults, error) {
			_, _, v := openVaultSearch(c, exec, prefs, endpoints, res)
			return openCLIVaults(v)
		},
		func(ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
			config := c
			config.ShouldRebuildIndex = options.Rebuild
			install := newPluginInstall(config, exec, endpoints, res)
			settingsDeps, searchDeps, vaultDeps := openVaultSearch(config, exec, prefs, endpoints, res)
			ports, err := vaultRegistry(vaultDeps.RegistryPath)
			if err != nil {
				return cli.Indexed{}, err
			}
			db, err := openIndex(vaultDeps, ctx)
			if err != nil {
				return cli.Indexed{}, err
			}
			r.closers = append(r.closers, db.Close)
			index := cliIndex{run: r, db: db, report: options.ErrorHandler}
			return newCLIIndexed(index, ports, install, settingsDeps, searchDeps, vaultDeps), nil
		},
		func() cli.PluginCommands {
			install := newPluginInstall(c, exec, endpoints, res)
			settingsDeps, searchDeps, vaultDeps := openVaultSearch(c, exec, prefs, endpoints, res)
			commands := getPluginCommands(install, settingsDeps, vaultDeps)
			return getCLIPluginCommands(r, c, commands, install, searchDeps, vaultDeps)
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

type cliIndex struct {
	run    *cliRun
	db     *Index
	report window.ErrorHandler
}

func newCLIIndexed(index cliIndex, ports registry.Ports, install pluginInstall, settingsDeps settingsWiring, searchDeps searchWiring, vaultDeps vaultWiring) cli.Indexed {
	db := index.db
	find := registry.NewFind(ports.Lookup).Execute
	unregister := vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults())
	prepareScan := func(ctx context.Context, v domain.Vault) (indexing.ReadWholeVault, error) {
		ranking, storage := openIndexedVault(searchDeps, vaultDeps, db)
		return prepareCLIScan(index, ctx, v, install, ranking, storage)
	}
	prepareSearch := func(ctx context.Context, v domain.Vault) (searchusecase.Search, error) {
		ranking, storage := openIndexedVault(searchDeps, vaultDeps, db)
		return prepareCLISearch(index, ctx, v, install, ranking, storage)
	}
	prepareQuery := func(ctx context.Context, v domain.Vault) (*searchqueryusecase.FindSearchResults, error) {
		ranking, storage := openIndexedVault(searchDeps, vaultDeps, db)
		_, err := openPluginWork(install, storage, db, v, index.run.tasks, index.run.models)
		if err != nil {
			return nil, err
		}
		return prepareCLIQuery(index, ctx, install, settingsDeps, ranking, storage)
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
		OperationResults: newClearOperationResults(db),
	}
}

func prepareCLIScan(i cliIndex, ctx context.Context, v domain.Vault, install pluginInstall, searchDeps searchWiring, vaultDeps vaultWiring) (indexing.ReadWholeVault, error) {
	db, r := i.db, i.run
	_, err := openPluginWork(install, vaultDeps, db, v, r.tasks, r.models)
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

func prepareCLISearch(i cliIndex, ctx context.Context, v domain.Vault, install pluginInstall, searchDeps searchWiring, vaultDeps vaultWiring) (searchusecase.Search, error) {
	db, r := i.db, i.run
	errorHandler := i.report
	_, err := openPluginWork(install, vaultDeps, db, v, r.tasks, r.models)
	if err != nil {
		return searchusecase.Search{}, err
	}
	embedder, why := openCLIEmbedder(r, searchDeps, ctx)
	reportSearchFallback(errorHandler, why)
	return newSearch(searchDeps, db, embedder, errorHandler, nil), nil
}

func prepareCLIQuery(
	i cliIndex, ctx context.Context, install pluginInstall, settingsDeps settingsWiring, searchDeps searchWiring, vaultDeps vaultWiring,
) (*searchqueryusecase.FindSearchResults, error) {
	db, r := i.db, i.run
	errorHandler := i.report
	embedder, why := openCLIEmbedder(r, searchDeps, ctx)
	reportSearchFallback(errorHandler, why)
	setModelCalls(install, settingsDeps, vaultDeps, run.NewConnectionNoticeTasks(r.tasks).Execute)
	return newFindSearchResults(openRerankDeps(install, settingsDeps, vaultDeps), settingsDeps, searchDeps, db, embedder, errorHandler), nil
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

func getCLIPluginCommands(r *cliRun, c Config, commands PluginCommandUseCases, install pluginInstall, searchDeps searchWiring, vaultDeps vaultWiring) cli.PluginCommands {
	find := func(nameOrPath string) (domain.Vault, error) { return findCLIVault(c, nameOrPath) }
	return cli.PluginCommands{
		List:   commands.List,
		ByName: plugincommand.NewRunByName(find),
		Open: func(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
			closeWork, err := openCLIPluginWork(r, ctx, v, install, searchDeps, vaultDeps)
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

func openCLIPluginWork(r *cliRun, ctx context.Context, v domain.Vault, install pluginInstall, searchDeps searchWiring, vaultDeps vaultWiring) (func() error, error) {
	db, err := openIndex(vaultDeps, ctx)
	if err != nil {
		return nil, err
	}
	_, storage := openIndexedVault(searchDeps, vaultDeps, db)
	pluginWork, err := openPluginWork(install, storage, db, v, r.tasks, r.models)
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
