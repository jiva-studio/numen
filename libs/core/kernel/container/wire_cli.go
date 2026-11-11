package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func openCLIVaults(c Config) (cli.Vaults, error) {
	ports, err := c.Registry()
	if err != nil {
		return cli.Vaults{}, err
	}
	unregister := func(db *Index) vault.Unregister {
		return vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults())
	}
	return cli.Vaults{
		Find:       registry.NewFind(ports.Lookup),
		Add:        vault.NewAdd(vaultIdentity(vaultOf(c)), ports.Vaults, ports.Lookup, ports.Saver, c.Clock()),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, vaultReaders(vaultOf(c))),
		RecordOpen: vault.NewRecordOpen(ports.Recorder),
		Now:        c.Clock().Now,
		OpenRename: func(ctx context.Context) (cli.Renaming, error) {
			return openWithIndex(c, ctx, func(db *Index) cli.Renaming {
				return cli.Renaming{Rename: vault.NewRename(ports.Vaults, ports.Saver, db.Vaults()), Close: db.Close}
			})
		},
		OpenUnregister: func(ctx context.Context) (cli.Unregistering, error) {
			return openWithIndex(c, ctx, func(db *Index) cli.Unregistering {
				return cli.Unregistering{Unregister: unregister(db), Close: db.Close}
			})
		},
		OpenErase: func(ctx context.Context) (cli.Erasing, error) {
			return openWithIndex(c, ctx, func(db *Index) cli.Erasing {
				erase := vault.NewErase(vaultIdentity(vaultOf(c)), vaultTrash(vaultOf(c)), unregister(db))
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
	return openWithIndex(c, ctx, func(db *Index) cli.Links {
		return cli.Links{Show: Queries(db).Refer.Links, Close: db.Close}
	})
}

func openCLIScan(c Config, ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
	indexing := c
	indexing.ShouldRebuildIndex = rebuild
	db, err := openIndex(vaultOf(indexing), ctx)
	if err != nil {
		return cli.Scan{}, err
	}
	scan, err := readCLIScan(indexing, ctx, db, v)
	if err != nil {
		return cli.Scan{}, errors.Join(err, db.Close())
	}
	return scan, nil
}

func readCLIScan(c Config, ctx context.Context, db *Index, v domain.Vault) (cli.Scan, error) {
	opened, err := openedPluginWork(c, db, v)
	if err != nil {
		return cli.Scan{}, err
	}
	embedder, closeEmbedder, why := searchEmbedder(searchOf(opened), ctx)
	read, err := ReadWholeVault(opened, ctx, db, embedder, v)
	if err != nil {
		return cli.Scan{}, errors.Join(err, closeIfOpen(closeEmbedder))
	}
	return cli.Scan{
		Read: read,
		Summary: func(ctx context.Context) (domain.VaultSummary, error) {
			return db.NoteQueries().GetSummary(ctx, v.ID)
		},
		Unembedded: why,
		Close: func() error {
			return errors.Join(closeIfOpen(closeEmbedder), db.Close())
		},
	}, nil
}

func openCLISearch(c Config,
	ctx context.Context, v domain.Vault, errorHandler window.ErrorHandler,
) (cli.Search, error) {
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return cli.Search{}, err
	}
	asking, err := openedPluginWork(c, db, v)
	if err != nil {
		return cli.Search{}, errors.Join(err, db.Close())
	}
	embedder, closeEmbedder, why := searchEmbedder(searchOf(asking), ctx)
	return cli.Search{
		Search: newSearch(searchOf(asking), db, embedder, errorHandler),
		Words:  why,
		Close: func() error {
			return errors.Join(closeIfOpen(closeEmbedder), db.Close())
		},
	}, nil
}

func openCLIProblems(c Config, ctx context.Context) (cli.Problems, error) {
	return openWithIndex(c, ctx, func(db *Index) cli.Problems {
		return cli.Problems{Problems: getProblems(vaultOf(c), db.NoteProblems()), Close: db.Close}
	})
}

func getCLIPluginCommands(c Config) cli.PluginCommands {
	return cli.PluginCommands{List: getPluginCommands(plugins(c)).List, Open: func(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
		return openPluginCommands(c, ctx, v)
	}}
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
		Scan: func(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
			return openCLIScan(c, ctx, v, rebuild)
		},
		Search: func(ctx context.Context, v domain.Vault, errorHandler window.ErrorHandler) (cli.Search, error) {
			return openCLISearch(c, ctx, v, errorHandler)
		},
		Problems:       func(ctx context.Context) (cli.Problems, error) { return openCLIProblems(c, ctx) },
		PluginCommands: func() cli.PluginCommands { return getCLIPluginCommands(c) },
	}
}
