package settingsport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// PluginSettings changes and checks one setting of a plugin.
type PluginSettings interface {
	UpdatePluginSetting(ctx context.Context, pluginID, key string, value any) error
	CheckPluginSetting(pluginID, key string, value any) error
}

// PluginConfig hands the plugins the settings the user keeps for them.
type PluginConfig interface {
	SetSettings(all map[string]map[string]any)
	SetPerformanceProfile(ctx context.Context, profile string)
	SetPersistHandler(handler plugin.PersistHandler)
}

// PluginHost is the plugin host as the use cases see it: the settings it changes and the settings it is handed.
type PluginHost interface {
	PluginSettings
	PluginConfig
}

// SecretRefresh configures the running plugins with the secrets as they stand now.
type SecretRefresh func(ctx context.Context) error

// ProfileResolver returns the performance profile the machine runs a profile setting as.
type ProfileResolver func(profile string) string
