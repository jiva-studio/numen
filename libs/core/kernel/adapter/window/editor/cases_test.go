package editor_test

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func openRegisteredWindow(cfg testInstallation, ctx context.Context, named string, out io.Writer) (*container.OpenedWindow, *container.WindowUseCases, error) {
	installation, err := cfg.OpenWindow(cfg.Execution, cfg.Preferences, cfg.Endpoints, cfg.Resources, ctx, container.WindowOpening{Named: named, Out: out})
	if err != nil {
		return nil, nil, err
	}
	return installation, &installation.UseCases, nil
}

type testInstallation struct {
	container.Config
	Execution   *container.PluginExecution
	Preferences *container.PluginPreferences
	Endpoints   *container.PluginEndpoints
	*container.Resources
}

func newTestInstallation(c container.Config) testInstallation {
	execution, prefs, endpoints := container.PreparePluginHost(nil, nil, nil, c.SafeMode)
	return testInstallation{Config: c, Execution: execution, Preferences: prefs, Endpoints: endpoints, Resources: container.NewResources()}
}
