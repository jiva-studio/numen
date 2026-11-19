package main

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func configureLocations(cfg container.Config, host *container.PluginHost, res *container.Resources, where cli.Locations) container.Config {
	if where.Index != "" {
		cfg.IndexPath = where.Index
	}
	cfg.RegistryPath = where.Registry
	cfg.ServiceDir = where.ServiceDir
	cfg.SafeMode = where.SafeMode
	host.Runtime.SetSafeMode(cfg.SafeMode)
	if applied, _, err := cfg.ApplySettings(host, res); err == nil {
		cfg = applied
	}
	return cfg
}

func makeDeps(cfg container.Config, host *container.PluginHost, res *container.Resources) func(cli.Locations) cli.Session {
	return func(where cli.Locations) cli.Session {
		cfg = configureLocations(cfg, host, res, where)

		return container.OpenCLI(cfg, host, res)
	}
}
