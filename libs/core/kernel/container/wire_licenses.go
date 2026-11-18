package container

import (
	"context"

	windowlicense "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/license"
	licensingadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/licensing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/license"
)

func openLicenseUseCases(c pluginsWiring) *windowlicense.UseCases {
	if c.PluginHost.Views.Declarations == nil {
		return nil
	}
	adapter := licensingadapter.NewAdapter(func(ctx context.Context) {
		if err := c.PluginHost.Configuration.Refresh(ctx); err != nil {
			return
		}
	})
	deps := license.NewDeps(
		c.PluginHost.Views.Declarations,
		c.KeyStore,
		adapter,
		adapter,
		c.vault.Clock,
		adapter,
	)
	return &windowlicense.UseCases{
		List:       license.NewList(deps),
		Activate:   license.NewActivate(deps),
		Deactivate: license.NewDeactivate(deps),
		Watch:      license.NewWatch(deps),
	}
}
