package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func (c Config) openCLIVaults() (cli.Vaults, error) {
	ports, err := c.Registry()
	if err != nil {
		return cli.Vaults{}, err
	}
	unregister := func(db *vault.Index) vault.Unregister {
		return vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults())
	}
	return cli.Vaults{
		Find:       registry.NewFind(ports.Lookup),
		Add:        vault.NewAdd(c.VaultIdentity(), ports.Vaults, ports.Lookup, ports.Saver, c.Clock()),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, c.VaultReaders()),
		RecordOpen: vault.NewRecordOpen(ports.Recorder),
		Now:        c.Clock().Now,
		OpenRename: func(ctx context.Context) (cli.Renaming, error) {
			db, err := c.OpenIndex(ctx)
			if err != nil {
				return cli.Renaming{}, err
			}
			return cli.Renaming{Rename: vault.NewRename(ports.Vaults, ports.Saver, db.Vaults()), Close: db.Close}, nil
		},
		OpenUnregister: func(ctx context.Context) (cli.Unregistering, error) {
			db, err := c.OpenIndex(ctx)
			if err != nil {
				return cli.Unregistering{}, err
			}
			return cli.Unregistering{Unregister: unregister(db), Close: db.Close}, nil
		},
		OpenErase: func(ctx context.Context) (cli.Erasing, error) {
			db, err := c.OpenIndex(ctx)
			if err != nil {
				return cli.Erasing{}, err
			}
			erase := vault.NewErase(c.VaultIdentity(), c.Trash(), unregister(db))
			return cli.Erasing{Erase: erase, Close: db.Close}, nil
		},
	}, nil
}

func (c Config) openCLILinks(ctx context.Context) (cli.Links, error) {
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return cli.Links{}, err
	}
	notes := c.Notes(db, c.UpdateIndex(db))
	return cli.Links{Show: notes.Refer.Links, Close: db.Close}, nil
}

func (c Config) openCLIScan(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
	indexing := c
	indexing.ShouldRebuildIndex = rebuild
	db, err := indexing.OpenIndex(ctx)
	if err != nil {
		return cli.Scan{}, err
	}
	indexing, err = indexing.openedPluginWork(ctx, db, v)
	if err != nil {
		return cli.Scan{}, errors.Join(err, db.Close())
	}
	embedder, closeEmbedder, why := indexing.embedder(ctx)
	read, err := indexing.ReadWholeVault(ctx, db, embedder, v)
	if err != nil {
		return cli.Scan{}, errors.Join(err, closeIfOpen(closeEmbedder), db.Close())
	}
	return cli.Scan{
		Read: read,
		Summary: func(ctx context.Context) (domain.VaultSummary, error) {
			return db.Summaries().GetSummary(ctx, v.ID)
		},
		Unembedded: why,
		Close: func() error {
			return errors.Join(closeIfOpen(closeEmbedder), db.Close())
		},
	}, nil
}

func (c Config) openCLISearch(
	ctx context.Context, v domain.Vault, errorHandler window.ErrorHandler,
) (cli.Search, error) {
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return cli.Search{}, err
	}
	asking, err := c.openedPluginWork(ctx, db, v)
	if err != nil {
		return cli.Search{}, errors.Join(err, db.Close())
	}
	embedder, closeEmbedder, why := asking.embedder(ctx)
	return cli.Search{
		Search: asking.NewSearch(db, embedder, errorHandler),
		Words:  why,
		Close: func() error {
			return errors.Join(closeIfOpen(closeEmbedder), db.Close())
		},
	}, nil
}

func (c Config) openCLIProblems(ctx context.Context) (cli.Problems, error) {
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return cli.Problems{}, err
	}
	return cli.Problems{Problems: c.GetProblems(db.Problems()), Close: db.Close}, nil
}

func (c Config) getCLIPluginCommands() cli.PluginCommands {
	return cli.PluginCommands{List: c.getPluginCommands().List, Open: c.OpenPluginCommands}
}

func closeIfOpen(closeFn func() error) error {
	if closeFn == nil {
		return nil
	}
	return closeFn()
}

// GetVaultList builds the use case that lists the vaults of the registry given.
func (c Config) GetVaultList(ports registry.Ports) registry.List {
	return registry.NewList(ports.Vaults)
}
