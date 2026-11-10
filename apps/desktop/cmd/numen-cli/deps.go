package main

import (
	"context"

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
			Vaults: cfg.OpenCLIVaults,
			Links:  cfg.OpenCLILinks,
			Scan: func(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
				return cfg.OpenCLIScan(ctx, v, rebuild)
			},
			Search: func(ctx context.Context, v domain.Vault, errorHandler window.ErrorHandler) (cli.Search, error) {
				return cfg.OpenCLISearch(ctx, v, errorHandler)
			},
			Problems:       cfg.OpenCLIProblems,
			PluginCommands: cfg.GetCLIPluginCommands,
		}
	}
}
