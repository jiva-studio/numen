package settingsport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

type PluginSettings interface {
	CheckPluginSetting(pluginID, key string, value any) error
}

// PluginConfig hands the plugins the settings the user keeps for them.
type PluginConfig interface {
	SetSettings(all map[string]map[string]any)
	SetPerformanceProfile(ctx context.Context, profile string)
	SetPersistHandler(handler plugin.PersistHandler)
}

// PluginSettingApplier makes a plugin follow one setting that changed.
type PluginSettingApplier func(ctx context.Context, pluginID, key string, value any) error

// PluginSettingValues keeps the settings of the plugins in memory.
type PluginSettingValues interface {
	// ReplaceSetting stores the value. It returns the value it replaced, whether there was one, and
	// whether the dependencies the plugin declares are others under the new value.
	ReplaceSetting(pluginID, key string, value any) (previous any, hadPrevious, changesDependencies bool)
	// RestoreSetting puts back the value ReplaceSetting replaced, or removes the setting where there was none.
	RestoreSetting(pluginID, key string, previous any, hadPrevious bool)
	// FillGeneratedSettings gives each enabled plugin a value for the settings it asks the host to generate.
	FillGeneratedSettings()
}

type PluginStopper interface {
	StopPlugin(ctx context.Context, pluginID string) error
}

// FailureClearer forgets the failure of a plugin, which makes it startable again.
type FailureClearer interface {
	ClearFailure(pluginID string)
}

// PluginListeners binds the ports the settings of a plugin ask for.
type PluginListeners interface {
	// SyncListeners binds the ports and closes the ones no longer asked for. A port that cannot be
	// bound fails the plugin and is returned as the error.
	SyncListeners(ctx context.Context, pluginID string) error
	// RefreshListeners syncs the listeners and reports an error that is not already the failure of the plugin.
	RefreshListeners(ctx context.Context, pluginID string)
}

// PluginConfigPush sends the configuration to a running plugin.
type PluginConfigPush interface {
	NotifyPluginConfig(ctx context.Context, pluginID string) error
}

// SecretRefresh configures the running plugins with the secrets as they stand now.
type SecretRefresh func(ctx context.Context) error

// ProfileResolver returns the performance profile the machine runs a profile setting as.
type ProfileResolver func(profile string) string
