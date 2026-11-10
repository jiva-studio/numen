// Package pluginlend lends the capabilities of a plugin host to a container, for the tests of
// packages outside it.
package pluginlend

import (
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
)

// Lend lends the capabilities of host.
func Lend(host *plugin.Host) *container.PluginHost {
	return &container.PluginHost{
		PluginActivator:       host,
		PluginDiscoverer:      host,
		PluginResolver:        host,
		PluginSettingDeclarer: host,
		PluginEventEmitter:    host,
		SecretConfigurator:    host,
		SettingsPush:          host,
		SettingUpdater:        host,
		PersistHook:           host,
		PluginFailures:        host,
		PluginBinding:         host,
		DependencyReporter:    host,
		SetCompleter:          host.SetCompleter,
		SetPluginWork:         host.SetPluginWork,
		SetPluginVault:        host.SetPluginVault,
		GetPluginVault:        host.GetPluginVault,
		SetPluginTexts:        host.SetPluginTexts,
		SetPluginSearch:       host.SetPluginSearch,
		SetPluginWindow:       host.SetPluginWindow,
		SetPluginVaultList:    host.SetPluginVaultList,
		ContentTypeTable:      host,
		SetStorage: func(cacheRoot, dataRoot string) {
			host.SetStorageRoots(cacheRoot, dataRoot)
			if models := host.GetModelsDir(); models != "" {
				host.SetDependencies(container.GetDependencyRegistry(models))
			}
		},
		RefreshDependencies: func() { host.SetDependencies(container.GetDependencyRegistry("")) },
	}
}
