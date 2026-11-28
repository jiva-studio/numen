package container

import (
	"context"

	windowlicense "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/license"
	licensingadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/licensing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/license"
)

func openLicenseUseCases(install pluginInstall, settingsDeps settingsParts, vaultDeps vaultParts) *windowlicense.UseCases {
	if install.Licenses == nil {
		return nil
	}
	adapter := licensingadapter.NewAdapter(func(ctx context.Context) {
		if err := install.Configuration.Refresh(ctx); err != nil {
			settingsDeps.HandleError(err)
		}
	})
	deps := license.NewDeps(install.Licenses, install.Keys.Store, adapter, adapter, vaultDeps.Clock, adapter).WithErrorReporter(settingsDeps.HandleError)
	return &windowlicense.UseCases{List: license.NewList(deps), Activate: license.NewActivate(deps), Deactivate: license.NewDeactivate(deps), Watch: license.NewWatch(deps)}
}
