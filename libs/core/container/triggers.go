package container

import (
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/trigger"
)

func (c Config) openTriggers() *trigger.Triggers {
	if c.PluginHost == nil || c.SafeMode {
		return nil
	}
	return trigger.NewTriggers(
		commandbridge.Open(c.PluginHost), c.VaultReaders(), c.PluginHost,
		derivedfile.TriggerRecords{Options: c.vaultOptions(), Folder: coreFolder},
		c.getWhenConfig(),
	)
}

// whenConfig is the settings of the application and of the plugins the host resolved, as when
// conditions read them.
type whenConfig struct {
	config Config
}

func (c Config) getWhenConfig() whenConfig {
	return whenConfig{config: c}
}

// GetWhenConfig returns the config built from the settings the host resolved.
func (w whenConfig) GetWhenConfig() map[string]any {
	var plugins []pluginport.PluginInfo
	if w.config.PluginHost != nil {
		plugins = w.config.PluginHost.GetPluginInfos()
	}
	return pluginport.NewWhenConfig(w.config.listAppSettings(), plugins)
}
