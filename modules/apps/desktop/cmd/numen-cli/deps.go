package main

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// deps is what the terminal works through, built for the places one run was
// pointed at. An installation is assembled by the application, so the openers
// are made here and the terminal is handed them.
//
// Each one opens when its command asks and not before: a person listing vaults
// waits for no index, and one asking about a scan waits for no transcriber.
func deps(cfg container.Config) func(cli.Locations) cli.Deps {
	return func(where cli.Locations) cli.Deps {
		cfg.IndexPath = where.Index
		cfg.RegistryPath = where.Registry
		cfg.ServiceDir = where.ServiceDir

		return cli.Deps{
			Vaults: func() (cli.Vaults, error) {
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
			},

			Links: func(ctx context.Context) (cli.Links, error) {
				db, err := cfg.OpenIndex(ctx)
				if err != nil {
					return cli.Links{}, err
				}
				notes := cfg.Notes(
					db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), cfg.Level(db))
				return cli.Links{Show: notes.Links, Close: db.Close}, nil
			},

			Scan: func(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
				reading := cfg
				reading.RebuildIndex = rebuild
				db, err := reading.OpenIndex(ctx)
				if err != nil {
					return cli.Scan{}, err
				}
				// A terminal waits for the model rather than making the vectors
				// later, and a vault is read whether or not one arrives.
				embedder, closeEmbedder, why := reading.Embedder(ctx)
				read, err := reading.ReadWholeVault(ctx, db, embedder, v)
				if err != nil {
					if closeEmbedder != nil {
						_ = closeEmbedder()
					}
					_ = db.Close()
					return cli.Scan{}, err
				}
				return cli.Scan{
					Read: read,
					Summary: func(ctx context.Context) (domain.VaultSummary, error) {
						return db.Queries().Summary(ctx, v.ID)
					},
					Unembedded: why,
					Close: func() error {
						if closeEmbedder != nil {
							_ = closeEmbedder()
						}
						return db.Close()
					},
				}, nil
			},

			Recognise: func(
				ctx context.Context, v domain.Vault, fetching func(),
			) (cli.Recognise, error) {
				if !cfg.RecogniserReady() {
					fetching()
				}
				models, closeModels, why := cfg.Recogniser(ctx)
				if why != nil {
					return cli.Recognise{}, why
				}
				db, err := cfg.OpenIndex(ctx)
				if err != nil {
					_ = closeModels()
					return cli.Recognise{}, err
				}
				cut, err := cfg.Extract(db.Sources(), db.SourcesKnown(), v)
				if err != nil {
					_ = closeModels()
					_ = db.Close()
					return cli.Recognise{}, err
				}
				return cli.Recognise{
					Recognise: cfg.Recognise(db.Sources(), models),
					Cut:       cut,
					Close: func() error {
						_ = closeModels()
						return db.Close()
					},
				}, nil
			},

			Search: func(ctx context.Context, trouble port.Trouble) (cli.Search, error) {
				db, err := cfg.OpenIndex(ctx)
				if err != nil {
					return cli.Search{}, err
				}
				// Only the provider that embeds questions is opened. Nothing a
				// search does fills an index.
				asking, closeAsking, why := cfg.Asking(ctx)
				return cli.Search{
					Search: cfg.Searching(db, asking, trouble),
					Words:  why,
					Close: func() error {
						if closeAsking != nil {
							_ = closeAsking()
						}
						return db.Close()
					},
				}, nil
			},

			Problems: func(ctx context.Context) (cli.Problems, error) {
				db, err := cfg.OpenIndex(ctx)
				if err != nil {
					return cli.Problems{}, err
				}
				return cli.Problems{Problems: db.Problems(), Close: db.Close}, nil
			},
		}
	}
}
