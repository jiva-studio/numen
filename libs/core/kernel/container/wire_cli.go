package container

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// CLISession is what one run of the command line opens: the index, the embedders and the plugin work of the
// command it runs. The plugin work of the run reports into one list of tasks, and Close hands back the ones
// that failed.
type CLISession struct {
	c       Config
	tasks   *run.Tasks
	models  *run.Lock
	closers []func() error
}

// OpenCLI returns the session of one run of the command line over the installation c describes. It opens
// nothing: each command opens what it works over.
func OpenCLI(c Config) *CLISession {
	return &CLISession{c: c, tasks: run.NewTasks(), models: &run.Lock{}}
}

// OpenVaults returns the use cases over the list of vaults.
func (s *CLISession) OpenVaults() (cli.Vaults, error) {
	a := plugins(s.c)
	ports, err := s.c.registry()
	if err != nil {
		return cli.Vaults{}, err
	}
	find := registry.NewFind(ports.Lookup).Execute
	return cli.Vaults{
		Add:        vault.NewAdd(vaultIdentity(a.vault), ports.Vaults, ports.Lookup, ports.Saver, s.c.Clock()),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, vaultReaders(a.vault)),
		RecordOpen: vault.NewRecordOpenByName(find, vault.NewRecordOpen(ports.Recorder)),
	}, nil
}

// OpenIndexed opens the index, which Close closes, and returns the use cases over it.
func (s *CLISession) OpenIndexed(ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
	c := s.c
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
	s.closers = append(s.closers, db.Close)
	find := registry.NewFind(ports.Lookup).Execute
	unregister := vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults())
	over := cliOver{session: s, config: c, db: db}
	return cli.Indexed{
		Rename:     vault.NewRenameByName(find, vault.NewRename(ports.Vaults, ports.Saver, db.Vaults())),
		Unregister: vault.NewUnregisterByName(find, unregister),
		Erase:      vault.NewEraseByName(find, vault.NewErase(vaultIdentity(a.vault), vaultTrash(a.vault), unregister)),
		Scan: cli.Scan{
			Scan: indexing.NewScanVault(indexing.NewReadWholeVaultByName(find, over.prepareScan), db.NoteQueries()),
			Now:  c.Clock().Now,
		},
		Search: cli.Search{
			Search: searchusecase.NewByName(find, over.searchPreparing(options.ErrorHandler).prepare),
		},
		Links:    query.NewShowLinksByName(find, Queries(db).Refer.Links),
		Problems: vault.NewGetProblemsByName(find, getProblems(a.vault, db.NoteProblems())),
	}, nil
}

// GetPluginCommands returns the commands the enabled plugins declare, and what runs one over a vault.
func (s *CLISession) GetPluginCommands() cli.PluginCommands {
	return cli.PluginCommands{
		List:   getPluginCommands(plugins(s.c)).List,
		ByName: plugincommand.NewRunByName(s.findVault),
		Open:   s.openPluginCommands,
	}
}

// Close closes what the run opened, the embedders before the index, and returns the tasks of the plugin work
// that failed, one line each.
func (s *CLISession) Close() ([]string, error) {
	var err error
	for _, closeOne := range slices.Backward(s.closers) {
		err = errors.Join(err, closeOne())
	}
	return listFailedTasks(s.tasks.List()), err
}

func (s *CLISession) findVault(nameOrPath string) (domain.Vault, error) {
	ports, err := s.c.registry()
	if err != nil {
		return domain.Vault{}, err
	}
	return registry.NewFind(ports.Lookup).Execute(nameOrPath)
}

func (s *CLISession) openPluginCommands(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
	db, err := openIndex(vaultOf(s.c), ctx)
	if err != nil {
		return plugincommand.Run{}, nil, err
	}
	_, pluginWork, err := openPluginWork(s.c, db, v, s.tasks, s.models)
	if err != nil {
		return plugincommand.Run{}, nil, errors.Join(err, db.Close())
	}
	return getPluginCommands(plugins(s.c)).Run, closeAfterWork(ctx, pluginWork, db.Close), nil
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

// cliOver is the index one command of a run works over, with the configuration it was opened under.
type cliOver struct {
	session *CLISession
	config  Config
	db      *Index
}

func (o cliOver) prepareScan(ctx context.Context, v domain.Vault) (indexing.ReadWholeVault, error) {
	opened, err := o.openPluginWork(v)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	embedder, why := o.session.openEmbedder(opened, ctx)
	read, err := readWholeVault(opened.search, ctx, o.db, embedder, v, vaultMakeRefresh(opened.vault, o.db, o.db.Notes()))
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	read.Unembedded = why
	return read, nil
}

func (o cliOver) openPluginWork(v domain.Vault) (pluginsWiring, error) {
	opened, _, err := openPluginWork(o.config, o.db, v, o.session.tasks, o.session.models)
	return opened, err
}

// cliSearching prepares the search of one vault over the index of a run, and tells errorHandler that the search
// runs on words alone.
type cliSearching struct {
	over         cliOver
	errorHandler func(error)
}

func (o cliOver) searchPreparing(errorHandler func(error)) cliSearching {
	return cliSearching{over: o, errorHandler: errorHandler}
}

func (p cliSearching) prepare(ctx context.Context, v domain.Vault) (searchusecase.Search, error) {
	asking, err := p.over.openPluginWork(v)
	if err != nil {
		return searchusecase.Search{}, err
	}
	embedder, why := p.over.session.openEmbedder(asking, ctx)
	if why != nil && p.errorHandler != nil {
		p.errorHandler(why)
	}
	return newSearch(asking.search, p.over.db, embedder, p.errorHandler), nil
}

// openEmbedder opens the embedder of the plugins a, which Close closes, and returns why there is none.
func (s *CLISession) openEmbedder(a pluginsWiring, ctx context.Context) (embeddingport.Embedder, error) {
	embedder, closeEmbedder, why := searchEmbedder(a.search, ctx)
	if closeEmbedder != nil {
		s.closers = append(s.closers, closeEmbedder)
	}
	return embedder, why
}
