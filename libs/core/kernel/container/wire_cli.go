package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func openCLIVaults(c Config) (cli.Vaults, error) {
	ports, err := c.registry()
	if err != nil {
		return cli.Vaults{}, err
	}
	find := registry.NewFind(ports.Lookup).Execute
	unregister := func(db *Index) vault.Unregister {
		return vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults()).WithFind(find)
	}
	return cli.Vaults{
		Add:        vault.NewAdd(vaultIdentity(vaultOf(c)), ports.Vaults, ports.Lookup, ports.Saver, c.Clock()),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, vaultReaders(vaultOf(c))),
		RecordOpen: vault.NewRecordOpen(ports.Recorder).WithFind(find),
		OpenRename: func(ctx context.Context) (cli.Renaming, error) {
			return openWithIndex(c, ctx, func(db *Index) cli.Renaming {
				return cli.Renaming{Rename: vault.NewRename(ports.Vaults, ports.Saver, db.Vaults()).WithFind(find), Close: db.Close}
			})
		},
		OpenUnregister: func(ctx context.Context) (cli.Unregistering, error) {
			return openWithIndex(c, ctx, func(db *Index) cli.Unregistering {
				return cli.Unregistering{Unregister: unregister(db), Close: db.Close}
			})
		},
		OpenErase: func(ctx context.Context) (cli.Erasing, error) {
			return openWithIndex(c, ctx, func(db *Index) cli.Erasing {
				erase := vault.NewErase(vaultIdentity(vaultOf(c)), vaultTrash(vaultOf(c)), unregister(db)).WithFind(find)
				return cli.Erasing{Erase: erase, Close: db.Close}
			})
		},
	}, nil
}

func openWithIndex[T any](c Config, ctx context.Context, build func(db *Index) T) (T, error) {
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		var none T
		return none, err
	}
	return build(db), nil
}

func openCLILinks(c Config, ctx context.Context) (cli.Links, error) {
	ports, err := c.registry()
	if err != nil {
		return cli.Links{}, err
	}
	find := registry.NewFind(ports.Lookup).Execute
	return openWithIndex(c, ctx, func(db *Index) cli.Links {
		return cli.Links{Show: Queries(db).Refer.Links.WithFind(find), Close: db.Close}
	})
}

func openCLIScan(c Config, ctx context.Context, rebuild bool) (cli.Scan, error) {
	ports, err := c.registry()
	if err != nil {
		return cli.Scan{}, err
	}
	indexingCfg := c
	indexingCfg.ShouldRebuildIndex = rebuild
	db, err := openIndex(vaultOf(indexingCfg), ctx)
	if err != nil {
		return cli.Scan{}, err
	}
	var closeEmbedder func() error
	find := registry.NewFind(ports.Lookup).Execute
	prepare := func(ctx context.Context, v domain.Vault) (indexing.ReadWholeVault, error) {
		return prepareCLIScan(indexingCfg, ctx, db, v, &closeEmbedder)
	}
	return cli.Scan{
		Read: indexing.NewReadWholeVaultByName(find, prepare),
		Summary: func(ctx context.Context, v domain.Vault) (domain.VaultSummary, error) {
			return db.NoteQueries().GetSummary(ctx, v.ID)
		},
		Now: c.Clock().Now,
		Close: func() error {
			return errors.Join(closeIfOpen(closeEmbedder), db.Close())
		},
	}, nil
}

func prepareCLIScan(
	c Config, ctx context.Context, db *Index, v domain.Vault, closeEmbedder *func() error,
) (indexing.ReadWholeVault, error) {
	opened, err := openedPluginWork(c, db, v)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	var embedder embeddingport.Embedder
	var why error
	embedder, *closeEmbedder, why = searchEmbedder(searchOf(opened), ctx)
	read, err := ReadWholeVault(opened, ctx, db, embedder, v)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	read.Unembedded = why
	return read, nil
}

func openCLISearch(c Config,
	ctx context.Context, errorHandler window.ErrorHandler,
) (cli.Search, error) {
	ports, err := c.registry()
	if err != nil {
		return cli.Search{}, err
	}
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return cli.Search{}, err
	}
	var closeEmbedder func() error
	find := registry.NewFind(ports.Lookup).Execute
	prepare := func(ctx context.Context, v domain.Vault) (searchusecase.Search, error) {
		return prepareCLISearch(c, ctx, db, v, errorHandler, &closeEmbedder)
	}
	return cli.Search{
		Search: searchusecase.NewByName(find, prepare),
		Close: func() error {
			return errors.Join(closeIfOpen(closeEmbedder), db.Close())
		},
	}, nil
}

func prepareCLISearch(
	c Config, ctx context.Context, db *Index, v domain.Vault,
	errorHandler window.ErrorHandler, closeEmbedder *func() error,
) (searchusecase.Search, error) {
	asking, err := openedPluginWork(c, db, v)
	if err != nil {
		return searchusecase.Search{}, err
	}
	var embedder embeddingport.Embedder
	var why error
	embedder, *closeEmbedder, why = searchEmbedder(searchOf(asking), ctx)
	reportSearchFallback(errorHandler, why)
	return newSearch(searchOf(asking), db, embedder, errorHandler), nil
}

func reportSearchFallback(errorHandler window.ErrorHandler, err error) {
	if err != nil && errorHandler != nil {
		errorHandler(err)
	}
}

func openCLIProblems(c Config, ctx context.Context) (cli.Problems, error) {
	ports, err := c.registry()
	if err != nil {
		return cli.Problems{}, err
	}
	find := registry.NewFind(ports.Lookup).Execute
	return openWithIndex(c, ctx, func(db *Index) cli.Problems {
		return cli.Problems{Problems: getProblems(vaultOf(c), db.NoteProblems()).WithFind(find), Close: db.Close}
	})
}

func getCLIPluginCommands(c Config) cli.PluginCommands {
	find := func(nameOrPath string) (domain.Vault, error) {
		ports, err := c.registry()
		if err != nil {
			return domain.Vault{}, err
		}
		return registry.NewFind(ports.Lookup).Execute(nameOrPath)
	}
	return cli.PluginCommands{
		List: getPluginCommands(plugins(c)).List.WithFind(find),
		Open: func(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
			return openPluginCommands(c, ctx, v)
		},
	}
}

func closeIfOpen(closeFn func() error) error {
	if closeFn == nil {
		return nil
	}
	return closeFn()
}

// OpenCLI returns what the command line calls to reach the vaults, the scan, the search, the links, the
// problems and the plugin commands of the installation c describes.
func OpenCLI(c Config) cli.Deps {
	return cli.Deps{
		Vaults: func() (cli.Vaults, error) { return openCLIVaults(c) },
		Links:  func(ctx context.Context) (cli.Links, error) { return openCLILinks(c, ctx) },
		Scan: func(ctx context.Context, rebuild bool) (cli.Scan, error) {
			return openCLIScan(c, ctx, rebuild)
		},
		Search: func(ctx context.Context, errorHandler window.ErrorHandler) (cli.Search, error) {
			return openCLISearch(c, ctx, errorHandler)
		},
		Problems:       func(ctx context.Context) (cli.Problems, error) { return openCLIProblems(c, ctx) },
		PluginCommands: func() cli.PluginCommands { return getCLIPluginCommands(c) },
	}
}
