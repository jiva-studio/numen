package main

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/container"
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
		}
	}
}
