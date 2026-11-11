package embedderbridge_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func updateSetting(ctx context.Context, host *container.PluginRoles, pluginID, key string, value any) error {
	return settingsusecase.NewApplyPluginSetting(settingsusecase.ApplyPluginSettingDeps{
		Values: host.Settings, Stopper: host.Teardown, Failures: host.Failures, Listeners: host.Listeners, Push: host.Config,
	}).Execute(ctx, pluginID, key, value)
}
