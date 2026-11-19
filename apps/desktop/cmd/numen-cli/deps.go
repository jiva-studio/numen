package main

import (
	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func configureLocations(cfg container.Config, execution *container.PluginExecution, preferences *container.PluginPreferences, res *container.Resources, where cli.Locations) container.Config {
	if where.Index != "" {
		cfg.IndexPath = where.Index
	}
	cfg.RegistryPath = where.Registry
	cfg.ServiceDir = where.ServiceDir
	cfg.SafeMode = where.SafeMode
	execution.Runtime.SetSafeMode(cfg.SafeMode)
	if applied, _, err := cfg.ApplySettings(execution, preferences, res); err == nil {
		cfg = applied
	}
	return cfg
}

func makeDeps(cfg container.Config, services platform.Installation) func(cli.Locations) cli.Session {
	return func(where cli.Locations) cli.Session {
		cfg = configureLocations(cfg, services.Execution, services.Preferences, services.Resources, where)

		return container.OpenCLI(cfg, services.Execution, services.Preferences, services.Endpoints, services.Resources)
	}
}
