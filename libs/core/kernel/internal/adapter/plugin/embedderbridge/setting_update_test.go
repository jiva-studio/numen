package embedderbridge

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func updateSetting(ctx context.Context, host *pluginhost.Host, pluginID, key string, value any) error {
	return settingsusecase.NewApplyPluginSetting(settingsusecase.ApplyPluginSettingDeps{
		Values: host.Settings, Stopper: host.Teardown, Failures: host.Failures, Listeners: host.Listeners, Push: host.Config,
	}).Execute(ctx, pluginID, key, value)
}
