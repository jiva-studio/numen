package container

import (
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestlist"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/whenconfig"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/trigger"
)

func (c Config) openTriggers() *trigger.Triggers {
	if c.PluginHost == nil || c.SafeMode || c.TextVersions == nil {
		return nil
	}
	return trigger.NewTriggers(
		manifestlist.Open(c.PluginHost), c.GetPluginCommands().Run, c.VaultReaders(), c.PluginHost,
		c.TextVersions, c.getWhenConfig(),
	)
}

func (c Config) getWhenConfig() pluginport.WhenConfig {
	settings := whenconfig.Settings{App: c.listAppSettings}
	if c.PluginHost != nil {
		settings.Plugins = c.PluginHost.GetPluginInfos
	}
	return settings
}
