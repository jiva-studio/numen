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
	unregister := func(db *vault.Index) vault.Unregister {
		return vault.NewUnregister(ports.Vaults, ports.Remover, db.Stores.Vaults)
	}
	return cli.Vaults{
		Find:       registry.NewFind(ports.Lookup),
		Add:        vault.NewAdd(vaultIdentity(vaultOf(c)), ports.Vaults, ports.Lookup, ports.Saver, c.Clock()),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, vaultReaders(vaultOf(c))),
		RecordOpen: vault.NewRecordOpen(ports.Recorder),
		Now:        c.Clock().Now,
		OpenRename: func(ctx context.Context) (cli.Renaming, error) {
			db, err := openIndex(vaultOf(c), ctx)
			if err != nil {
				return cli.Renaming{}, err
			}
			return cli.Renaming{Rename: vault.NewRename(ports.Vaults, ports.Saver, db.Stores.Vaults), Close: db.Close}, nil
		},
		OpenUnregister: func(ctx context.Context) (cli.Unregistering, error) {
			db, err := openIndex(vaultOf(c), ctx)
			if err != nil {
				return cli.Unregistering{}, err
			}
			return cli.Unregistering{Unregister: unregister(db), Close: db.Close}, nil
		},
		OpenErase: func(ctx context.Context) (cli.Erasing, error) {
			db, err := openIndex(vaultOf(c), ctx)
			if err != nil {
				return cli.Erasing{}, err
			}
			erase := vault.NewErase(vaultIdentity(vaultOf(c)), vaultTrash(vaultOf(c)), unregister(db))
			return cli.Erasing{Erase: erase, Close: db.Close}, nil
		},
	}, nil
}

func openCLILinks(c Config, ctx context.Context) (cli.Links, error) {
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return cli.Links{}, err
	}
	notes := Notes(c, db, updateIndex(vaultOf(c), db))
	return cli.Links{Show: notes.Refer.Links, Close: db.Close}, nil
}

func openCLIScan(c Config, ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
	indexing := c
	indexing.ShouldRebuildIndex = rebuild
	db, err := openIndex(vaultOf(indexing), ctx)
	if err != nil {
		return cli.Scan{}, err
	}
	indexing, err = openedPluginWork(indexing, db, v)
	if err != nil {
		return cli.Scan{}, errors.Join(err, db.Close())
	}
	embedder, closeEmbedder, why := searchEmbedder(searchOf(indexing), ctx)
	read, err := ReadWholeVault(indexing, ctx, db, embedder, v)
	if err != nil {
		return cli.Scan{}, errors.Join(err, closeIfOpen(closeEmbedder), db.Close())
	}
	return cli.Scan{
		Read: read,
		Summary: func(ctx context.Context) (domain.VaultSummary, error) {
			return db.NoteQueries.Summaries.GetSummary(ctx, v.ID)
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
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return cli.Problems{}, err
	}
	return cli.Problems{Problems: getProblems(vaultOf(c), db.NoteQueries.Problems), Close: db.Close}, nil
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

// GetVaultList builds the use case that lists the vaults of the registry given.
func GetVaultList(_ Config, ports registry.Ports) registry.List {
	return registry.NewList(ports.Vaults)
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
