package container

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/operationresult"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"

	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"

	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// cliRun is what the parts of one run of the command line share: the one list of tasks the plugin work of
// the run reports into, and what Close closes.
type cliRun struct {
	tasks   *run.Tasks
	closers []func() error
}

// OpenCLI returns the session of one run of the command line over the installation c describes. It opens
// nothing: each command opens what it works over, and Close closes it.
func OpenCLI(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources) cli.Session {
	exec = getPluginExecution(exec)
	endpoints = getPluginEndpoints(endpoints)
	r := &cliRun{tasks: run.NewTasks()}
	return cli.NewSession(
		func() (cli.Vaults, error) {
			_, _, v := buildVaultSearch(c, exec, prefs, res, nil)
			return openCLIVaults(v)
		},
		func(ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
			config := c
			config.ShouldRebuildIndex = options.Rebuild
			install := newPluginInstall(config, prefs, endpoints, res)
			settingsDeps, plan, vaultDeps := buildVaultSearch(config, exec, prefs, res, nil)
			ports, db, err := config.openRegistryIndex(res, ctx)
			if err != nil {
				return cli.Indexed{}, err
			}
			index := newCLIIndex(r, db, options.ErrorHandler, newCLIPluginWork(install, exec, r))
			index.source = newSourceBinding(exec, config.SafeMode, install.Endpoints, res)
			index.workers = config.machine.ResolvedCompute.Workers
			prepare := newCLIPreparations(index, install, exec, settingsDeps, plan, vaultDeps)
			return newCLIIndexed(ports, db, vaultDeps, prepare, run.NewConnectionNoticeTasks(r.tasks).Execute), nil
		},
		func() cli.PluginCommands {
			install := newPluginInstall(c, prefs, endpoints, res)
			settingsDeps, plan, vaultDeps := buildVaultSearch(c, exec, prefs, res, nil)
			commands := newPluginCommands(getWhenConfig(settingsDeps, exec.Runtime.Listing), vaultDeps, exec, nil)
			plugins := getRunningExecution(exec, c.SafeMode).Runtime.Listing
			return getCLIPluginCommands(commands, newCLIPluginWork(install, exec, r), plugins, plan.ChunkPolicy, vaultDeps)
		},
		func() ([]string, error) { return closeCLI(r) },
	)
}

func openCLIVaults(vaultDeps vaultWiring) (cli.Vaults, error) {
	ports, err := vaultRegistry(vaultDeps.RegistryPath)
	if err != nil {
		return cli.Vaults{}, err
	}
	return cli.Vaults{
		Add:        newAddVault(vaultIdentity(vaultDeps.Options), vaultDeps.Clock, ports),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, vaultDeps.Readers),
		RecordOpen: vault.NewRecordOpenByName(ports.Lookup, vault.NewRecordOpen(ports.Recorder)),
	}, nil
}

type cliPluginWork func(vaultWiring, *vaultIndex, domain.Vault) (*run.PluginWork, error)

func newCLIPluginWork(install pluginInstall, exec *PluginExecution, r *cliRun) cliPluginWork {
	return func(vaultDeps vaultWiring, db *vaultIndex, v domain.Vault) (*run.PluginWork, error) {
		exec.Runtime.pluginBinding.SetVaultPath(v.Path)
		if err := installCurrentVault(install, vaultDeps, db, v); err != nil {
			return nil, err
		}
		work := pluginsOpenPluginWork(install, exec, r.tasks)
		setPluginTexts(install.Endpoints.Lender, vaultDeps.TextVersions)
		return work, nil

	}
}

type cliIndex struct {
	source    sourcebridge.Binding
	workers   int
	openWork  cliPluginWork
	closers   *[]func() error
	db        *vaultIndex
	report    window.ErrorHandler
	plan      vaultIndexPlan
	vaultDeps vaultWiring
	plugins   textversion.TextPlugins
}

func newCLIIndex(r *cliRun, db *vaultIndex, report window.ErrorHandler, openWork cliPluginWork) cliIndex {
	r.closers = append(r.closers, db.Close)
	return cliIndex{
		db: db, report: report, openWork: openWork,
		closers: &r.closers,
	}
}

type cliPreparations struct {
	scan   func(context.Context, domain.Vault) (indexing.ReadWholeVault, error)
	search func(context.Context, domain.Vault) (searchusecase.Search, error)
	query  func(context.Context, domain.Vault, connection.ReportSetAside) (*searchqueryusecase.FindSearchResults, error)
}

func newCLIPreparations(index cliIndex, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, plan vaultIndexPlan, vaultDeps vaultWiring) cliPreparations {
	index.plan, index.vaultDeps = plan, vaultDeps
	index.plugins = getRunningExecution(exec, install.SafeMode).Runtime.Listing
	prepareQuery := func(ctx context.Context, v domain.Vault, notices connection.ReportSetAside) (*searchqueryusecase.FindSearchResults, error) {
		ranking, storage := openIndexedVault(plan, vaultDeps, index.db, index.plugins)
		embedder, err := openCLISearchSource(index, ctx, v, storage)
		if err != nil {
			return nil, err
		}
		setModelCalls(install, exec, settingsDeps, storage, notices)
		return newFindSearchResults(openQueryPorts(install, exec, settingsDeps, storage), settingsDeps, newTextSearch(ranking, index.workers), index.db, embedder, index.report), nil
	}
	return cliPreparations{scan: index.prepareScan, search: index.prepareSearch, query: prepareQuery}
}

func newCLIIndexed(ports registry.Ports, db *vaultIndex, vaultDeps vaultWiring, prepare cliPreparations, notices connection.ReportSetAside) cli.Indexed {
	find := ports.Lookup
	change := vaultVaults(vaultDeps, ports, db).Change
	return cli.Indexed{
		Changes: cli.VaultChanges{
			Rename:     vault.NewRenameByName(find, change.Rename),
			Unregister: vault.NewUnregisterByName(find, change.Unregister),
			Erase:      vault.NewEraseByName(find, change.Erase),
		},
		Scan: cli.Scan{
			Scan: indexing.NewScanVault(indexing.NewReadWholeVaultByName(find, prepare.scan), db.NoteQueries()),
			Now:  vaultDeps.Clock.Now,
		},
		Search:   cli.Search{Search: searchusecase.NewByName(find, prepare.search)},
		Links:    query.NewShowLinksByName(find, vaultQueries(db).Refer.Links),
		Problems: vault.NewGetProblemsByName(find, getProblems(db.NoteProblems())),
		Query: searchqueryusecase.NewFindSearchResultsByName(find, func(ctx context.Context, v domain.Vault) (*searchqueryusecase.FindSearchResults, error) {
			return prepare.query(ctx, v, notices)
		}),
		OperationResults: operationresult.NewClearOperationResults(db.OperationResults(operationResultRows)),
	}
}

func (i cliIndex) prepareScan(ctx context.Context, v domain.Vault) (indexing.ReadWholeVault, error) {
	plan, vaultDeps := openIndexedVault(i.plan, i.vaultDeps, i.db, i.plugins)
	if _, err := i.openWork(vaultDeps, i.db, v); err != nil {
		return indexing.ReadWholeVault{}, err
	}
	embedder, why := openCLIEmbedder(i.closers, i.source, ctx)
	read, err := readWholeVault(plan, ctx, i.db, embedder.searchSource, v, vaultMakeRefresh(vaultDeps, i.db, i.db.Notes()))
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	read.Unembedded = why
	return read, nil
}

func (i cliIndex) prepareSearch(ctx context.Context, v domain.Vault) (searchusecase.Search, error) {
	plan, vaultDeps := openIndexedVault(i.plan, i.vaultDeps, i.db, i.plugins)
	embedder, err := openCLISearchSource(i, ctx, v, vaultDeps)
	if err != nil {
		return searchusecase.Search{}, err
	}
	return newSearch(newTextSearch(plan, i.workers), i.db, embedder.searchSource, i.report, nil), nil
}

// openCLISearchSource opens the plugin work over v and the source a search asks, and reports why that source cannot answer.
func openCLISearchSource(i cliIndex, ctx context.Context, v domain.Vault, vaultDeps vaultWiring) (sourceAccess, error) {
	if _, err := i.openWork(vaultDeps, i.db, v); err != nil {
		return sourceAccess{}, err
	}
	embedder, why := openCLIEmbedder(i.closers, i.source, ctx)
	reportSearchFallback(i.report, why)
	return embedder, nil
}

func reportSearchFallback(errorHandler window.ErrorHandler, err error) {
	if err != nil && errorHandler != nil {
		errorHandler(err)
	}
}

func openCLIEmbedder(closers *[]func() error, source sourcebridge.Binding, ctx context.Context) (sourceAccess, error) {
	embedder, closeEmbedder, why := source.Open(ctx)
	if closeEmbedder != nil {
		*closers = append(*closers, closeEmbedder)
	}
	return getSourceAccess(embedder), why
}

func getCLIPluginCommands(commands pluginCommandUseCases, openWork cliPluginWork, plugins textversion.TextPlugins, chunks noteChunkPolicy, vaultDeps vaultWiring) cli.PluginCommands {
	ports, err := vaultRegistry(vaultDeps.RegistryPath)
	find := ports.Lookup
	if err != nil {
		find = appstate.UnopenedRegistry{Err: err}
	}
	return cli.PluginCommands{
		List:   commands.List,
		ByName: plugincommand.NewRunByName(find),
		Open: func(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
			closeWork, err := openCLIPluginWork(openWork, ctx, v, plugins, chunks, vaultDeps)
			if err != nil {
				return plugincommand.Run{}, nil, err
			}
			return commands.Run, closeWork, nil
		},
	}
}

func openCLIPluginWork(openWork cliPluginWork, ctx context.Context, v domain.Vault, plugins textversion.TextPlugins, chunks noteChunkPolicy, vaultDeps vaultWiring) (func() error, error) {
	db, err := openIndex(vaultDeps, ctx)
	if err != nil {
		return nil, err
	}
	storage := openVaultTextVersions(vaultDeps, db, plugins, chunks)
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
