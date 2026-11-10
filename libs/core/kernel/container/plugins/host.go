package plugins

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/completionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// Host is the plugin host as the capabilities the wiring and the use cases consume. NewHost builds
// every capability from one plugin host, and a Config holds the one *Host the container made: the
// settings subscription is keyed by that pointer.
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
	pluginport.PluginTextHost
	pluginport.PluginSearchHost
	pluginport.PluginWindowHost
	pluginport.PluginVaultListHost
	pluginport.DependencyReporter
	completionport.CompleterHost
	contenttypeport.ContentTypeTable
	host *plugin.Host
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
		PluginTextHost:        host,
		PluginSearchHost:      host,
		PluginWindowHost:      host,
		PluginVaultListHost:   host,
		DependencyReporter:    host,
		CompleterHost:         host,
		ContentTypeTable:      host,
		host:                  host,
	}
}

// SetStorage sets the folders the plugins store files in, and keeps the dependencies in the models
// folder the host names, where it names one.
func (h *Host) SetStorage(cacheRoot, dataRoot string) {
	h.host.SetStorageRoots(cacheRoot, dataRoot)
	// gomutants:disable-next-line
	if models := h.host.GetModelsDir(); models != "" {
		// gomutants:disable-next-line
		h.host.SetDependencies(NewDependenciesIn(models))
	}
}

// RefreshDependencies provisions the dependencies as the environment names them now.
func (h *Host) RefreshDependencies() {
	h.host.SetDependencies(NewDependenciesIn(""))
}
