package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func openCLIRename(c Config, ctx context.Context, ports registry.Ports, find vault.VaultFinding) (cli.Renaming, error) {
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return cli.Renaming{}, err
	}
	rename := vault.NewRename(ports.Vaults, ports.Saver, db.Vaults())
	return cli.Renaming{Rename: vault.NewRenameByName(find, rename), Close: db.Close}, nil
}

func openCLIUnregister(c Config, ctx context.Context, ports registry.Ports, find vault.VaultFinding) (cli.Unregistering, error) {
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return cli.Unregistering{}, err
	}
	unregister := vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults())
	return cli.Unregistering{Unregister: vault.NewUnregisterByName(find, unregister), Close: db.Close}, nil
}

func openCLIErase(c Config, ctx context.Context, ports registry.Ports, find vault.VaultFinding) (cli.Erasing, error) {
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return cli.Erasing{}, err
	}
	unregister := vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults())
	erase := vault.NewErase(vaultIdentity(vaultOf(c)), vaultTrash(vaultOf(c)), unregister)
	return cli.Erasing{Erase: vault.NewEraseByName(find, erase), Close: db.Close}, nil
}

func openCLIVaults(c Config) (cli.Vaults, error) {
	ports, err := c.registry()
	if err != nil {
		return cli.Vaults{}, err
	}
	find := registry.NewFind(ports.Lookup).Execute
	return cli.Vaults{
		Add:        vault.NewAdd(vaultIdentity(vaultOf(c)), ports.Vaults, ports.Lookup, ports.Saver, c.Clock()),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, vaultReaders(vaultOf(c))),
		RecordOpen: vault.NewRecordOpenByName(find, vault.NewRecordOpen(ports.Recorder)),
		OpenRename: func(ctx context.Context) (cli.Renaming, error) { return openCLIRename(c, ctx, ports, find) },
		OpenUnregister: func(ctx context.Context) (cli.Unregistering, error) {
			return openCLIUnregister(c, ctx, ports, find)
		},
		OpenErase: func(ctx context.Context) (cli.Erasing, error) { return openCLIErase(c, ctx, ports, find) },
	}, nil
}

func openCLILinks(c Config, ctx context.Context) (cli.Links, error) {
	ports, err := c.registry()
	if err != nil {
		return cli.Links{}, err
	}
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return cli.Links{}, err
	}
	find := registry.NewFind(ports.Lookup).Execute
	return cli.Links{Show: query.NewShowLinksByName(find, Queries(db).Refer.Links), Close: db.Close}, nil
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
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return cli.Problems{}, err
	}
	find := registry.NewFind(ports.Lookup).Execute
	return cli.Problems{
		Problems: vault.NewGetProblemsByName(find, getProblems(vaultOf(c), db.NoteProblems())), Close: db.Close,
	}, nil
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
		List:   getPluginCommands(plugins(c)).List,
		ByName: plugincommand.NewRunByName(find),
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
