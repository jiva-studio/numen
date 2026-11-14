package none

import "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"

// PluginSettings accepts every setting of a plugin: no plugin host runs to refuse one.
type PluginSettings struct{}

func (PluginSettings) CheckPluginSetting(string, string, any) error { return nil }

// PluginInfos is the plugin list of an installation with no plugin host.
type PluginInfos struct{}

func (PluginInfos) GetPluginInfos() []plugin.Info { return nil }
