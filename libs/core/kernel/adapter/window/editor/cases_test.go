package editor_test

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func openRegisteredWindow(cfg testInstallation, ctx context.Context, named string, out io.Writer) (*container.OpenedWindow, *container.WindowUseCases, error) {
	installation, err := container.OpenWindow(cfg.Config, cfg.PluginHost, cfg.Resources, ctx, named, out)
	if err != nil {
		return nil, nil, err
	}
	return installation, &installation.UseCases, nil
}

type testInstallation struct {
	container.Config
	PluginHost *container.PluginHost
	*container.Resources
}

func newTestInstallation(c container.Config) testInstallation {
	return testInstallation{Config: c, PluginHost: container.PreparePluginHost(nil, c.SafeMode), Resources: container.NewResources()}
}
