package main

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func configureLocations(cfg container.Config, where cli.Locations) container.Config {
	if where.Index != "" {
		cfg.IndexPath = where.Index
	}
	cfg.RegistryPath = where.Registry
	cfg.ServiceDir = where.ServiceDir
	cfg.SafeMode = where.SafeMode
	cfg = container.NewConfig(cfg)
	if applied, _, err := cfg.ApplySettings(); err == nil {
		cfg = applied
	}
	return cfg
}

func makeDeps(cfg container.Config) func(cli.Locations) cli.Session {
	return func(where cli.Locations) cli.Session {
		cfg = configureLocations(cfg, where)

		return container.OpenCLI(cfg)
	}
}
