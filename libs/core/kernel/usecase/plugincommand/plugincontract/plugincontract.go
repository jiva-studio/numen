// Package plugincontract re-exports the plugin port contract.
package plugincontract

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

type (
	DependencyLookup      = pluginport.DependencyLookup
	DependencyReporter    = pluginport.DependencyReporter
	DependencyProvisioner = pluginport.DependencyProvisioner
	KeyStore              = pluginport.KeyStore
	PluginActivator       = pluginport.PluginActivator
	PluginCache           = pluginport.PluginCache
	PluginCommands        = pluginport.PluginCommands
	PluginDeclarations    = pluginport.PluginDeclarations
	PluginBinding         = pluginport.PluginBinding
	PluginFailures        = pluginport.PluginFailures
	PersistHook           = pluginport.PersistHook
	SecretConfigurator    = pluginport.SecretConfigurator
	SettingUpdater        = pluginport.SettingUpdater
	SettingsPush          = pluginport.SettingsPush
	PluginDiscoverer      = pluginport.PluginDiscoverer
	PluginEventEmitter    = pluginport.PluginEventEmitter
	PluginResolver        = pluginport.PluginResolver
	PluginStores          = pluginport.PluginStores
)
