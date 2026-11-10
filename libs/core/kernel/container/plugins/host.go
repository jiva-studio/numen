package plugins

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// Host is the plugin host as the capabilities the wiring and the use cases consume. NewHost builds
// every capability from one plugin host, and a Config holds the one *Host the container made: the
// settings subscription is keyed by that pointer. SetStorage sets the folders
// the plugins store files in and keeps the dependencies in the models folder the host names;
// RefreshDependencies provisions them as the environment names them now.
type Host struct {
	pluginport.PluginActivator
	pluginport.PluginDiscoverer
	pluginport.PluginResolver
	pluginport.PluginSettingDeclarer
	pluginport.PluginEventEmitter
	pluginport.SecretConfigurator
	pluginport.SettingsPush
	pluginport.SettingUpdater
	pluginport.PersistHook
	pluginport.PluginFailures
	pluginport.PluginBinding
	pluginport.DependencyReporter
	contenttypeport.ContentTypeTable
	SetCompleter        func(completer hostendpoint.Completer)
	SetPluginWork       func(work hostendpoint.WorkQueue)
	SetPluginVault      func(vault *hostendpoint.VaultAccess)
	GetPluginVault      func() *hostendpoint.VaultAccess
	SetPluginTexts      func(texts hostendpoint.TextVersions)
	SetPluginSearch     func(search *hostendpoint.SearchAccess)
	SetPluginWindow     func(window hostendpoint.WindowControl)
	SetPluginVaultList  func(list hostendpoint.VaultList)
	SetStorage          func(cacheRoot, dataRoot string)
	RefreshDependencies func()
}

// NewHost lends the capabilities of host.
func NewHost(host *plugin.Host) *Host {
	return &Host{
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
		SetStorage:            func(cacheRoot, dataRoot string) { setStorage(host, cacheRoot, dataRoot) },
		RefreshDependencies:   func() { host.SetDependencies(NewDependenciesIn("")) },
	}
}

func setStorage(host *plugin.Host, cacheRoot, dataRoot string) {
	host.SetStorageRoots(cacheRoot, dataRoot)
	// gomutants:disable-next-line
	if models := host.GetModelsDir(); models != "" {
		// gomutants:disable-next-line
		host.SetDependencies(NewDependenciesIn(models))
	}
}
