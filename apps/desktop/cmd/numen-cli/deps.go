package main

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

func configureLocations(cfg container.Config, where cli.Locations) container.Config {
	cfg.IndexPath = where.Index
	cfg.RegistryPath = where.Registry
	cfg.ServiceDir = where.ServiceDir
	cfg = cfg.SetSafeMode(where.SafeMode)
	if selected, err := cfg.Settings(); err == nil {
		cfg = cfg.SetSettings(selected)
	}
	return cfg
}

// makeDeps is what the terminal works through, built for the places one run was
// pointed at. An installation is assembled by the application, so the openers
// are made here and the terminal is handed them.
//
// Each one opens when its command asks and not before: a person listing vaults
// waits for no index, and one asking about a scan waits for no transcriber.
func makeDeps(cfg container.Config) func(cli.Locations) cli.Deps {
	return func(where cli.Locations) cli.Deps {
		cfg = configureLocations(cfg, where)

		return cli.Deps{
			Vaults: func() (cli.Vaults, error) { return createVaults(cfg) },
			Links:  func(ctx context.Context) (cli.Links, error) { return createLinks(ctx, cfg) },
			Scan: func(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
				cfg.SetPluginVaultPath(v.Path)
				return createScan(ctx, cfg, v, rebuild)
			},
			TextMenu: func(ctx context.Context, v domain.Vault) (cli.TextMenu, error) {
				return createTextMenu(ctx, cfg, v)
			},
			Search: func(ctx context.Context, errorHandler windowport.ErrorHandler) (cli.Search, error) {
				return createSearch(ctx, cfg, errorHandler)
			},
			Problems:       func(ctx context.Context) (cli.Problems, error) { return createProblems(ctx, cfg) },
			PluginCommands: func() cli.PluginCommands { return createPluginCommands(cfg) },
		}
	}
}

// createVaults is the registry of vaults with the readers over them. The index
// behind the rows is opened only when the rows are asked for.
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

// createLinks is the links of a note, read from the index.
func createLinks(ctx context.Context, cfg container.Config) (cli.Links, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Links{}, err
	}
	notes := cfg.Notes(
		db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), cfg.Level(db))
	return cli.Links{Show: notes.Links, Close: db.Close}, nil
}

// createScan is the walk of a vault into the index. The vault is read whether
// or not the embedding model arrives.
func createScan(
	ctx context.Context, cfg container.Config, v domain.Vault, rebuild bool,
) (cli.Scan, error) {
	cfg.SetPluginVaultPath(v.Path)
	indexing := cfg
	indexing.ShouldRebuildIndex = rebuild
	db, err := indexing.OpenIndex(ctx)
	if err != nil {
		return cli.Scan{}, err
	}
	embedder, closeEmbedder, why := indexing.Embedder(ctx)
	read, err := indexing.ReadWholeVault(ctx, db, embedder, v)
	if err != nil {
		closeIfOpen(closeEmbedder)
		_ = db.Close()
		return cli.Scan{}, err
	}
	return cli.Scan{
		Read: read,
		Summary: func(ctx context.Context) (domain.VaultSummary, error) {
			return db.Queries().GetSummary(ctx, v.ID)
		},
		Unembedded: why,
		Close: func() error {
			closeIfOpen(closeEmbedder)
			return db.Close()
		},
	}, nil
}

// closeIfOpen closes what a function returned to close, when it returned one.
func closeIfOpen(closeFn func() error) {
	if closeFn != nil {
		_ = closeFn()
	}
}

// createTextMenu is the menu the files of a vault offer about their text.
func createTextMenu(ctx context.Context, cfg container.Config, v domain.Vault) (cli.TextMenu, error) {
	opened, err := cfg.OpenTextMenu(ctx, v)
	if err != nil {
		return cli.TextMenu{}, err
	}
	return cli.TextMenu{Menu: opened.Menu, Tasks: opened.Tasks, Cut: opened.Cut, Close: opened.Close}, nil
}

// createSearch is the search over the index. Only the provider that embeds
// questions is opened, because nothing a search does fills an index.
func createSearch(
	ctx context.Context, cfg container.Config, errorHandler windowport.ErrorHandler,
) (cli.Search, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Search{}, err
	}
	asking, closeAsking, why := cfg.Embedder(ctx)
	return cli.Search{
		Search: cfg.NewSearch(db, asking, errorHandler),
		Words:  why,
		Close: func() error {
			closeIfOpen(closeAsking)
			return db.Close()
		},
	}, nil
}

// createProblems is the problems the index holds.
func createProblems(ctx context.Context, cfg container.Config) (cli.Problems, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Problems{}, err
	}
	return cli.Problems{Problems: db.Problems(), Close: db.Close}, nil
}

// createPluginCommands is the commands the enabled plugins declare.
func createPluginCommands(cfg container.Config) cli.PluginCommands {
	return cli.PluginCommands{List: cfg.GetPluginCommands().List, Open: cfg.OpenPluginCommands}
}
