package settings

import (
	"context"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

// PluginHealth is the health items of the plugins.
type PluginHealth interface {
	RemovePlugin(pluginID string)
}

// ApplyPluginSetting makes a plugin follow a setting that changed: it stores the value in memory,
// stops the plugin when its dependencies are others, and syncs its listeners and sends it its
// configuration. A listener that cannot be bound puts the previous value back. A plugin switched off
// loses its health items.
type ApplyPluginSetting struct {
	values    settingsport.PluginSettingValues
	stopper   settingsport.PluginStopper
	failures  settingsport.FailureClearer
	listeners settingsport.PluginListeners
	push      settingsport.PluginConfigPush
	health    PluginHealth
}

// ApplyPluginSettingDeps is what ApplyPluginSetting is made of.
type ApplyPluginSettingDeps struct {
	Values    settingsport.PluginSettingValues
	Stopper   settingsport.PluginStopper
	Failures  settingsport.FailureClearer
	Listeners settingsport.PluginListeners
	Push      settingsport.PluginConfigPush
	Health    PluginHealth
}

// NewApplyPluginSetting makes the use case.
func NewApplyPluginSetting(deps ApplyPluginSettingDeps) *ApplyPluginSetting {
	return &ApplyPluginSetting{
		values: deps.Values, stopper: deps.Stopper, failures: deps.Failures, listeners: deps.Listeners, push: deps.Push,
		health: deps.Health,
	}
}

func (u *ApplyPluginSetting) Execute(ctx context.Context, pluginID, key string, value any) error {
	previous, hadPrevious, changesDependencies := u.values.ReplaceSetting(pluginID, key, value)

	if changesDependencies && key != settingsdomain.HostFieldEnabled {
		if err := u.stopper.StopPlugin(ctx, pluginID); err != nil {
			return err
		}
	}

	if key == settingsdomain.HostFieldEnabled {
		if err := u.applyEnabledFlag(ctx, pluginID, value); err != nil {
			return err
		}
		return u.push.NotifyPluginConfig(ctx, pluginID)
	}
	if err := u.listeners.SyncListeners(ctx, pluginID); err != nil {
		u.values.RestoreSetting(pluginID, key, previous, hadPrevious)
		u.listeners.RefreshListeners(ctx, pluginID)
		return err
	}
	return u.push.NotifyPluginConfig(ctx, pluginID)
}

func (u *ApplyPluginSetting) applyEnabledFlag(ctx context.Context, pluginID string, value any) error {
	enabled, ok := value.(bool)
	if !ok {
		return nil
	}
	var stopErr error
	if !enabled {
		stopErr = u.stopper.StopPlugin(ctx, pluginID)
		u.health.RemovePlugin(pluginID)
	} else {
		u.values.FillGeneratedSettings()
	}
	u.failures.ClearFailure(pluginID)
	u.listeners.RefreshListeners(ctx, pluginID)
	return stopErr
}
