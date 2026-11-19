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
	buildIndexed := func(db *Index, rebuild bool) pluginsWiring {
		config := c
		config.ShouldRebuildIndex = rebuild
		return openIndexedPlugins(config, exec, prefs, endpoints, res, db)
	}
	return cli.NewSession(
		func() (cli.Vaults, error) { return openCLIVaults(plugins(c, exec, prefs, endpoints, res)) },
		func(ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
			config := c
			config.ShouldRebuildIndex = options.Rebuild
			return openCLIIndexed(r, plugins(config, exec, prefs, endpoints, res), buildIndexed, ctx, options)
		},
		func() cli.PluginCommands {
			return getCLIPluginCommands(r, c, plugins(c, exec, prefs, endpoints, res), buildIndexed)
		},
		func() ([]string, error) { return closeCLI(r) },
	)
}

func openCLIVaults(a pluginsWiring) (cli.Vaults, error) {
	ports, err := vaultRegistry(a.vault.RegistryPath)
	if err != nil {
		return cli.Vaults{}, err
	}
	find := registry.NewFind(ports.Lookup).Execute
	return cli.Vaults{
		Add:        vault.NewAdd(vaultIdentity(a.vault), ports.Vaults, ports.Lookup, ports.Saver, a.vault.Clock),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, vaultReaders(a.vault)),
		RecordOpen: vault.NewRecordOpenByName(find, vault.NewRecordOpen(ports.Recorder)),
	}, nil
}

func openCLIIndexed(r *cliRun, a pluginsWiring, buildIndexed func(*Index, bool) pluginsWiring, ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
	ports, err := vaultRegistry(a.vault.RegistryPath)
	if err != nil {
		return cli.Indexed{}, err
	}
	db, err := openIndex(a.vault, ctx)
	if err != nil {
		return cli.Indexed{}, err
	}
	r.closers = append(r.closers, db.Close)
	find := registry.NewFind(ports.Lookup).Execute
	unregister := vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults())
	prepareScan := func(ctx context.Context, v domain.Vault) (indexing.ReadWholeVault, error) {
		return prepareCLIScan(r, buildIndexed(db, options.Rebuild), db, ctx, v)
	}
	prepareSearch := func(ctx context.Context, v domain.Vault) (searchusecase.Search, error) {
		return prepareCLISearch(r, buildIndexed(db, options.Rebuild), db, options.ErrorHandler, ctx, v)
	}
	prepareQuery := func(ctx context.Context, v domain.Vault) (*searchqueryusecase.FindSearchResults, error) {
		return prepareCLIQuery(r, buildIndexed(db, options.Rebuild), db, options.ErrorHandler, ctx, v)
	}
	return cli.Indexed{
		Changes: cli.VaultChanges{
			Rename:     vault.NewRenameByName(find, vault.NewRename(ports.Vaults, ports.Saver, db.Vaults())),
			Unregister: vault.NewUnregisterByName(find, unregister),
			Erase:      vault.NewEraseByName(find, vault.NewErase(vaultIdentity(a.vault), vaultTrash(a.vault), unregister)),
		},
		Scan: cli.Scan{
			Scan: indexing.NewScanVault(indexing.NewReadWholeVaultByName(find, prepareScan), db.NoteQueries()),
			Now:  a.vault.Clock.Now,
		},
		Search:           cli.Search{Search: searchusecase.NewByName(find, prepareSearch)},
		Links:            query.NewShowLinksByName(find, Queries(db).Refer.Links),
		Problems:         vault.NewGetProblemsByName(find, getProblems(a.vault, db.NoteProblems())),
		Query:            searchqueryusecase.NewFindSearchResultsByName(find, prepareQuery),
		OperationResults: newClearOperationResults(db),
	}, nil
}

func prepareCLIScan(
	r *cliRun, a pluginsWiring, db *Index, ctx context.Context, v domain.Vault,
) (indexing.ReadWholeVault, error) {
	opened, _, err := openPluginWork(a, db, v, r.tasks, r.models)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	embedder, why := openCLIEmbedder(r, opened, ctx)
	read, err := readWholeVault(opened.indexing, ctx, db, embedder, v, vaultMakeRefresh(opened.vault, db, db.Notes()))
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	read.Unembedded = why
	return read, nil
}

func prepareCLISearch(
	r *cliRun, a pluginsWiring, db *Index, errorHandler window.ErrorHandler, ctx context.Context, v domain.Vault,
) (searchusecase.Search, error) {
	asking, _, err := openPluginWork(a, db, v, r.tasks, r.models)
	if err != nil {
		return searchusecase.Search{}, err
	}
	embedder, why := openCLIEmbedder(r, asking, ctx)
	reportSearchFallback(errorHandler, why)
	return newSearch(asking.search, db, embedder, errorHandler, nil), nil
}

func prepareCLIQuery(
	r *cliRun, a pluginsWiring, db *Index, errorHandler window.ErrorHandler, ctx context.Context, v domain.Vault,
) (*searchqueryusecase.FindSearchResults, error) {
	asking, _, err := openPluginWork(a, db, v, r.tasks, r.models)
	if err != nil {
		return nil, err
	}
	embedder, why := openCLIEmbedder(r, asking, ctx)
	reportSearchFallback(errorHandler, why)
	setModelCalls(asking, run.NewConnectionNoticeTasks(r.tasks).Execute)
	return newFindSearchResults(asking, db, embedder, errorHandler), nil
}

func reportSearchFallback(errorHandler window.ErrorHandler, err error) {
	if err != nil && errorHandler != nil {
		errorHandler(err)
	}
}

func openCLIEmbedder(r *cliRun, a pluginsWiring, ctx context.Context) (searchsourceport.Source, error) {
	embedder, closeEmbedder, why := searchSource(a.source, ctx)
	if closeEmbedder != nil {
		r.closers = append(r.closers, closeEmbedder)
	}
	return embedder, why
}

func getCLIPluginCommands(r *cliRun, c Config, a pluginsWiring, buildIndexed func(*Index, bool) pluginsWiring) cli.PluginCommands {
	find := func(nameOrPath string) (domain.Vault, error) { return findCLIVault(c, nameOrPath) }
	return cli.PluginCommands{
		List:   getPluginCommands(a).List,
		ByName: plugincommand.NewRunByName(find),
		Open: func(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
			return openCLIPluginCommands(r, a, buildIndexed, ctx, v)
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

func openCLIPluginCommands(
	r *cliRun, a pluginsWiring, buildIndexed func(*Index, bool) pluginsWiring, ctx context.Context, v domain.Vault,
) (plugincommand.Run, func() error, error) {
	db, err := openIndex(a.vault, ctx)
	if err != nil {
		return plugincommand.Run{}, nil, err
	}
	_, pluginWork, err := openPluginWork(buildIndexed(db, a.indexing.ShouldRebuildIndex), db, v, r.tasks, r.models)
	if err != nil {
		return plugincommand.Run{}, nil, errors.Join(err, db.Close())
	}
	return getPluginCommands(a).Run, run.CloseAfterWork(ctx, pluginWork, db.Close), nil
}

func closeCLI(r *cliRun) ([]string, error) {
	var err error
	for _, closeOne := range slices.Backward(r.closers) {
		err = errors.Join(err, closeOne())
	}
	return cli.FormatTaskFailures(r.tasks.List()), err
}
