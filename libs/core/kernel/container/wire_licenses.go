package container

import (
	"context"

	windowlicense "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/license"
	licensingadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/licensing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/license"
)

func openLicenseUseCases(install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) *windowlicense.UseCases {
	if settingsDeps.Views.Declarations == nil {
		return nil
	}
	adapter := licensingadapter.NewAdapter(func(ctx context.Context) {
		if err := settingsDeps.Configuration.Refresh(ctx); err != nil {
			settingsDeps.ErrorHandler(err)
		}
	})
	deps := license.NewDeps(settingsDeps.Views.Declarations, install.KeyStore, adapter, adapter, vaultDeps.Clock, adapter).WithErrorReporter(settingsDeps.ErrorHandler)
	return &windowlicense.UseCases{List: license.NewList(deps), Activate: license.NewActivate(deps), Deactivate: license.NewDeactivate(deps), Watch: license.NewWatch(deps)}
}
