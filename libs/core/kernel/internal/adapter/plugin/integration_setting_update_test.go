package plugin_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"

	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func updateSetting(ctx context.Context, host *pluginhost.Host, pluginID, key string, value any) error {
	return settingsusecase.NewApplyPluginSetting(settingsusecase.ApplyPluginSettingDeps{
		Values: host.Management.Settings, Stopper: host.Management.Teardown, Failures: host.Supervision.Failures, Listeners: host.Management.Listeners, Push: host.Management.Config,
		Health: host.Supervision.Health,
	}).Execute(ctx, pluginID, key, value)
}

func setEnabled(ctx context.Context, host *pluginhost.Host, pluginID string, enabled bool) error {
	return updateSetting(ctx, host, pluginID, "enabled", enabled)
}
