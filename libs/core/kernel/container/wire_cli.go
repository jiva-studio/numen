package container

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// cliRun is what the parts of one run of the command line share: the installation, the one list of tasks
// the plugin work of the run reports into, the lock of the models, and what Close closes.
type cliRun struct {
	c       Config
	tasks   *run.Tasks
	models  *run.Lock
	closers []func() error
}

// OpenCLI returns the session of one run of the command line over the installation c describes. It opens
// nothing: each command opens what it works over, and Close closes it.
func OpenCLI(c Config) cli.Session {
	r := &cliRun{c: c, tasks: run.NewTasks(), models: &run.Lock{}}
	return cli.NewSession(
		func() (cli.Vaults, error) { return openCLIVaults(r.c) },
		func(ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
			return openCLIIndexed(r, ctx, options)
		},
		func() cli.PluginCommands { return getCLIPluginCommands(r) },
		func() ([]string, error) { return closeCLI(r) },
	)
}

func openCLIVaults(c Config) (cli.Vaults, error) {
	a := plugins(c)
	ports, err := c.registry()
	if err != nil {
		return cli.Vaults{}, err
	}
	find := registry.NewFind(ports.Lookup).Execute
	return cli.Vaults{
		Add:        vault.NewAdd(vaultIdentity(a.vault), ports.Vaults, ports.Lookup, ports.Saver, c.Clock()),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, vaultReaders(a.vault)),
		RecordOpen: vault.NewRecordOpenByName(find, vault.NewRecordOpen(ports.Recorder)),
	}, nil
}

func openCLIIndexed(r *cliRun, ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
	c := r.c
	c.ShouldRebuildIndex = options.Rebuild
	a := plugins(c)
	ports, err := c.registry()
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
		return prepareCLIScan(r, c, db, ctx, v)
	}
	prepareSearch := func(ctx context.Context, v domain.Vault) (searchusecase.Search, error) {
		return prepareCLISearch(r, c, db, options.ErrorHandler, ctx, v)
	}
	return cli.Indexed{
		Rename:     vault.NewRenameByName(find, vault.NewRename(ports.Vaults, ports.Saver, db.Vaults())),
		Unregister: vault.NewUnregisterByName(find, unregister),
		Erase:      vault.NewEraseByName(find, vault.NewErase(vaultIdentity(a.vault), vaultTrash(a.vault), unregister)),
		Scan: cli.Scan{
			Scan: indexing.NewScanVault(indexing.NewReadWholeVaultByName(find, prepareScan), db.NoteQueries()),
			Now:  c.Clock().Now,
		},
		Search:   cli.Search{Search: searchusecase.NewByName(find, prepareSearch)},
		Links:    query.NewShowLinksByName(find, Queries(db).Refer.Links),
		Problems: vault.NewGetProblemsByName(find, getProblems(a.vault, db.NoteProblems())),
	}, nil
}

func prepareCLIScan(
	r *cliRun, c Config, db *Index, ctx context.Context, v domain.Vault,
) (indexing.ReadWholeVault, error) {
	opened, _, err := openPluginWork(c, db, v, r.tasks, r.models)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	embedder, why := openCLIEmbedder(r, opened, ctx)
	read, err := readWholeVault(opened.search, ctx, db, embedder, v, vaultMakeRefresh(opened.vault, db, db.Notes()))
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	read.Unembedded = why
	return read, nil
}

func prepareCLISearch(
	r *cliRun, c Config, db *Index, errorHandler window.ErrorHandler, ctx context.Context, v domain.Vault,
) (searchusecase.Search, error) {
	asking, _, err := openPluginWork(c, db, v, r.tasks, r.models)
	if err != nil {
		return searchusecase.Search{}, err
	}
	embedder, why := openCLIEmbedder(r, asking, ctx)
	reportSearchFallback(errorHandler, why)
	return newSearch(asking.search, db, embedder, errorHandler, nil), nil
}

func reportSearchFallback(errorHandler window.ErrorHandler, err error) {
	if err != nil && errorHandler != nil {
		errorHandler(err)
	}
}

func openCLIEmbedder(r *cliRun, a pluginsWiring, ctx context.Context) (searchsourceport.Source, error) {
	embedder, closeEmbedder, why := searchSource(a.search, ctx)
	if closeEmbedder != nil {
		r.closers = append(r.closers, closeEmbedder)
	}
	return embedder, why
}

func getCLIPluginCommands(r *cliRun) cli.PluginCommands {
	find := func(nameOrPath string) (domain.Vault, error) { return findCLIVault(r.c, nameOrPath) }
	return cli.PluginCommands{
		List:   getPluginCommands(plugins(r.c)).List,
		ByName: plugincommand.NewRunByName(find),
		Open: func(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
			return openCLIPluginCommands(r, ctx, v)
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

func openCLIPluginCommands(r *cliRun, ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
	db, err := openIndex(vaultOf(r.c), ctx)
	if err != nil {
		return plugincommand.Run{}, nil, err
	}
	_, pluginWork, err := openPluginWork(r.c, db, v, r.tasks, r.models)
	if err != nil {
		return plugincommand.Run{}, nil, errors.Join(err, db.Close())
	}
	return getPluginCommands(plugins(r.c)).Run, closeAfterWork(ctx, pluginWork, db.Close), nil
}

func closeCLI(r *cliRun) ([]string, error) {
	var err error
	for _, closeOne := range slices.Backward(r.closers) {
		err = errors.Join(err, closeOne())
	}
	return listFailedTasks(r.tasks.List()), err
}

func listFailedTasks(tasks []work.Task) []string {
	var failed []string
	for _, task := range tasks {
		if task.Error != "" {
			failed = append(failed, fmt.Sprintf("%s: %s: %s", getTaskOwner(task), task.Title, task.Error))
		}
	}
	return failed
}

func getTaskOwner(task work.Task) string {
	if task.PluginName != "" {
		return task.PluginName
	}
	return task.PluginID
}
