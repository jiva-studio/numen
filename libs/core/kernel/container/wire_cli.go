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
		func(ctx context.Context, options cli.IndexOptions) (cli.IndexRun, error) {
			config := c
			config.ShouldRebuildIndex = options.Rebuild
			install := newPluginInstall(config, prefs, endpoints, res)
			settingsDeps, plan, vaultDeps := buildVaultSearch(config, exec, prefs, res, nil)
			ports, err := openVaultRegistry(vaultDeps.RegistryPath)
			if err != nil {
				return cli.IndexRun{}, err
			}
			db, err := openIndex(vaultDeps, ctx)
			if err != nil {
				return cli.IndexRun{}, err
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
			return getCLIPluginCommands(c, commands, newCLIPluginWork(install, exec, r), plugins, plan.ChunkPolicy, vaultDeps)
		},
		func() ([]string, error) { return closeCLI(r) },
	)
}

func openCLIVaults(vaultDeps vaultParts) (cli.Vaults, error) {
	ports, err := openVaultRegistry(vaultDeps.RegistryPath)
	if err != nil {
		return cli.Vaults{}, err
	}
	return cli.Vaults{
		Add:        vault.NewAdd(createVaultIdentity(vaultDeps.Options), ports.Vaults, ports.Lookup, ports.Saver, vaultDeps.Clock),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, vaultDeps.Readers),
		RecordOpen: vault.NewRecordOpenByName(ports.Lookup, vault.NewRecordOpen(ports.Recorder)),
	}, nil
}

type cliPluginWork func(vaultParts, *vaultIndex, domain.Vault) (*run.PluginWork, error)

func newCLIPluginWork(install pluginInstall, exec *PluginExecution, r *cliRun) cliPluginWork {
	return func(vaultDeps vaultParts, db *vaultIndex, v domain.Vault) (*run.PluginWork, error) {
		exec.Runtime.pluginBinding.SetVaultPath(v.Path)
		if err := installCurrentVault(install, vaultDeps, db, v); err != nil {
			return nil, err
		}
		work := openPluginWork(install, exec, r.tasks)
		setPluginTexts(install.Endpoints.Lender, vaultDeps.TextVersions)
		return work, nil

	}
}

type cliIndex struct {
	source   sourcebridge.Binding
	workers  int
	openWork cliPluginWork
	closers  *[]func() error
	db       *vaultIndex
	report   window.ErrorHandler
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

func newCLIPreparations(index cliIndex, install pluginInstall, exec *PluginExecution, settingsDeps settingsParts, plan vaultIndexPlan, vaultDeps vaultParts) cliPreparations {
	plugins := getRunningExecution(exec, install.SafeMode).Runtime.Listing
	db := index.db
	prepareScan := func(ctx context.Context, v domain.Vault) (indexing.ReadWholeVault, error) {
		ranking, storage := openIndexedVault(plan, vaultDeps, db, plugins)
		return prepareCLIScan(index, ctx, v, ranking, storage)
	}
	prepareSearch := func(ctx context.Context, v domain.Vault) (searchusecase.Search, error) {
		ranking, storage := openIndexedVault(plan, vaultDeps, db, plugins)
		return prepareCLISearch(index, ctx, v, ranking, storage)
	}
	prepareQuery := func(ctx context.Context, v domain.Vault, notices connection.ReportSetAside) (*searchqueryusecase.FindSearchResults, error) {
		ranking, storage := openIndexedVault(plan, vaultDeps, db, plugins)
		_, err := index.openWork(storage, db, v)
		if err != nil {
			return nil, err
		}
		errorHandler := index.report
		embedder, why := openCLIEmbedder(index.closers, index.source, ctx)
		reportSearchFallback(errorHandler, why)
		setModelCalls(install, exec, settingsDeps, storage, notices)
		return newFindSearchResults(openQueryPorts(install, exec, settingsDeps, storage), settingsDeps, newTextSearch(ranking, index.workers), db, embedder, errorHandler), nil
	}
	return cliPreparations{scan: prepareScan, search: prepareSearch, query: prepareQuery}
}

func newCLIIndexed(ports registry.Ports, db *vaultIndex, vaultDeps vaultParts, prepare cliPreparations, notices connection.ReportSetAside) cli.IndexRun {
	find := ports.Lookup
	unregister := vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults())
	return cli.IndexRun{
		Changes: cli.VaultChanges{
			Rename:     vault.NewRenameByName(find, vault.NewRename(ports.Vaults, ports.Saver, db.Vaults())),
			Unregister: vault.NewUnregisterByName(find, unregister),
			Erase:      vault.NewEraseByName(find, vault.NewErase(createVaultIdentity(vaultDeps.Options), createTrash(vaultDeps.Clock), unregister)),
		},
		Scan: cli.Scan{
			Scan: indexing.NewScanVault(indexing.NewReadWholeVaultByName(find, prepare.scan), db.NoteQueries()),
			Now:  vaultDeps.Clock.Now,
		},
		Search:   cli.Search{Search: searchusecase.NewByName(find, prepare.search)},
		Links:    query.NewShowLinksByName(find, createQueryUseCases(db).Refer.Links),
		Problems: vault.NewGetProblemsByName(find, getProblems(db.NoteProblems())),
		Query: searchqueryusecase.NewFindSearchResultsByName(find, func(ctx context.Context, v domain.Vault) (*searchqueryusecase.FindSearchResults, error) {
			return prepare.query(ctx, v, notices)
		}),
		OperationResults: operationresult.NewClearOperationResults(db.GetOperationResults(operationResultRows)),
	}
}

func prepareCLIScan(i cliIndex, ctx context.Context, v domain.Vault, plan vaultIndexPlan, vaultDeps vaultParts) (indexing.ReadWholeVault, error) {
	db := i.db
	_, err := i.openWork(vaultDeps, db, v)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	embedder, why := openCLIEmbedder(i.closers, i.source, ctx)
	read, err := readWholeVault(plan, ctx, db, embedder.searchSource, v, createRefresh(vaultDeps, db, db.Notes()))
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	read.Unembedded = why
	return read, nil
}

func prepareCLISearch(i cliIndex, ctx context.Context, v domain.Vault, plan vaultIndexPlan, vaultDeps vaultParts) (searchusecase.Search, error) {
	db := i.db
	errorHandler := i.report
	_, err := i.openWork(vaultDeps, db, v)
	if err != nil {
		return searchusecase.Search{}, err
	}
	embedder, why := openCLIEmbedder(i.closers, i.source, ctx)
	reportSearchFallback(errorHandler, why)
	return newSearch(newTextSearch(plan, i.workers), db, embedder.searchSource, errorHandler, nil), nil
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

func getCLIPluginCommands(_ Config, commands pluginCommandUseCases, openWork cliPluginWork, plugins textversion.TextPlugins, chunks noteChunkPolicy, vaultDeps vaultParts) cli.PluginCommands {
	ports, err := openVaultRegistry(vaultDeps.RegistryPath)
	find := ports.Lookup
	if err != nil {
		find = appstate.UnopenedRegistry{Err: err}
	}
	return cli.PluginCommands{
		List:   commands.List,
		ByName: plugincommand.NewRunByName(find),
		Open: func(ctx context.Context, v domain.Vault) (plugincommand.Runner, func() error, error) {
			closeWork, err := openCLIPluginWork(openWork, ctx, v, plugins, chunks, vaultDeps)
			if err != nil {
				return plugincommand.Runner{}, nil, err
			}
			return commands.Run, closeWork, nil
		},
	}
}

func openCLIPluginWork(openWork cliPluginWork, ctx context.Context, v domain.Vault, plugins textversion.TextPlugins, chunks noteChunkPolicy, vaultDeps vaultParts) (func() error, error) {
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
