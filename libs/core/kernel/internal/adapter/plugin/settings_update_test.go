package plugin

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/pluginset"

	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func updateSetting(ctx context.Context, host *Host, pluginID, key string, value any) error {
	return settingsusecase.NewApplyPluginSetting(settingsusecase.ApplyPluginSettingDeps{
		Values: host.Settings, Stopper: host.Teardown, Failures: host.Failures, Listeners: host.Listeners, Push: host.Config,
	}).Execute(ctx, pluginID, key, value)
}

func setEnabled(ctx context.Context, host *Host, pluginID string, enabled bool) error {
	return updateSetting(ctx, host, pluginID, pluginset.SettingEnabled, enabled)
}
