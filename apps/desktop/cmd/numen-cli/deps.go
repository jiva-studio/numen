package main

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func configureLocations(cfg container.Config, where cli.Locations) container.Config {
	if where.Index != "" {
		cfg.IndexPath = where.Index
	}
	cfg.RegistryPath = where.Registry
	cfg.ServiceDir = where.ServiceDir
	cfg = cfg.SetSafeMode(where.SafeMode)
	if selected, err := cfg.Settings(); err == nil {
		cfg = cfg.SetSettings(selected)
	}
	return cfg
}

func makeDeps(cfg container.Config) func(cli.Locations) cli.Deps {
	return func(where cli.Locations) cli.Deps {
		cfg = configureLocations(cfg, where)

		return cli.Deps{
			Vaults: func() (cli.Vaults, error) { return createVaults(cfg) },
			Links:  func(ctx context.Context) (cli.Links, error) { return createLinks(ctx, cfg) },
			Scan: func(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
				return createScan(ctx, cfg, v, rebuild)
			},
			Search: func(ctx context.Context, v domain.Vault, errorHandler window.ErrorHandler) (cli.Search, error) {
				return createSearch(ctx, cfg, v, errorHandler)
			},
			Problems:       func(ctx context.Context) (cli.Problems, error) { return createProblems(ctx, cfg) },
			PluginCommands: func() cli.PluginCommands { return createPluginCommands(cfg) },
		}
	}
}

func createVaults(cfg container.Config) (cli.Vaults, error) {
	registry, err := cfg.Registry()
	if err != nil {
		return cli.Vaults{}, err
	}
	return cli.Vaults{
		Registry: registry,
		Readers:  cfg.VaultReaders(),
		Identity: cfg.VaultIdentity(),
		Trash:    cfg.Trash(),
		Now:      cfg.Clock(),
		Rows: func(ctx context.Context) (cli.VaultRows, error) {
			db, err := cfg.OpenIndex(ctx)
			if err != nil {
				return cli.VaultRows{}, err
			}
			return cli.VaultRows{Vaults: db.Vaults(), Close: db.Close}, nil
		},
	}, nil
}

func createLinks(ctx context.Context, cfg container.Config) (cli.Links, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Links{}, err
	}
	notes := cfg.Notes(db, cfg.UpdateIndex(db))
	return cli.Links{Show: notes.Links, Close: db.Close}, nil
}

func createScan(
	ctx context.Context, cfg container.Config, v domain.Vault, rebuild bool,
) (cli.Scan, error) {
	indexing := cfg
	indexing.ShouldRebuildIndex = rebuild
	db, err := indexing.OpenIndex(ctx)
	if err != nil {
		return cli.Scan{}, err
	}
	indexing, err = indexing.OpenPluginWork(ctx, db, v)
	if err != nil {
		return cli.Scan{}, errors.Join(err, db.Close())
	}
	embedder, closeEmbedder, why := indexing.Embedder(ctx)
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

func closeIfOpen(closeFn func() error) error {
	if closeFn == nil {
		return nil
	}
	return closeFn()
}

func createSearch(
	ctx context.Context, cfg container.Config, v domain.Vault, errorHandler window.ErrorHandler,
) (cli.Search, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Search{}, err
	}
	asking, err := cfg.OpenPluginWork(ctx, db, v)
	if err != nil {
		return cli.Search{}, errors.Join(err, db.Close())
	}
	embedder, closeEmbedder, why := asking.Embedder(ctx)
	return cli.Search{
		Search: asking.NewSearch(db, embedder, errorHandler),
		Words:  why,
		Close: func() error {
			return errors.Join(closeIfOpen(closeEmbedder), db.Close())
		},
	}, nil
}

func createProblems(ctx context.Context, cfg container.Config) (cli.Problems, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Problems{}, err
	}
	return cli.Problems{Problems: db.Problems(), Close: db.Close}, nil
}

func createPluginCommands(cfg container.Config) cli.PluginCommands {
	return cli.PluginCommands{List: cfg.GetPluginCommands().List, Open: cfg.OpenPluginCommands}
}
