// Package plugincontract re-exports the plugin port contract.
package plugincontract

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

type (
	DependencyCatalog      = pluginport.DependencyCatalog
	DependencyProvisioner  = pluginport.DependencyProvisioner
	KeyStore               = pluginport.KeyStore
	Lease                  = pluginport.Lease
	PluginActivator        = pluginport.PluginActivator
	PluginCache            = pluginport.PluginCache
	PluginCommands         = pluginport.PluginCommands
	PluginDeclarations     = pluginport.PluginDeclarations
	PluginConfigurator     = pluginport.PluginConfigurator
	PluginDiscoverer       = pluginport.PluginDiscoverer
	PluginEventEmitter     = pluginport.PluginEventEmitter
	PluginLifecycleManager = pluginport.PluginLifecycleManager
	PluginManager          = pluginport.PluginManager
	PluginRegistryManager  = pluginport.PluginRegistryManager
	PluginResolver         = pluginport.PluginResolver
	PluginSearch           = pluginport.PluginSearch
	PluginSearchIndex      = pluginport.PluginSearchIndex
	PluginPassageSearch    = pluginport.PluginPassageSearch
	PluginSearchPolicy     = pluginport.PluginSearchPolicy
	PluginStore            = pluginport.PluginStore
	PluginStores           = pluginport.PluginStores
	PluginTextHost         = pluginport.PluginTextHost
	PluginSearchHost       = pluginport.PluginSearchHost
	PluginTexts            = pluginport.PluginTexts
	PluginVault            = pluginport.PluginVault
	PluginVaultLifecycle   = pluginport.PluginVaultLifecycle
	PluginWindow           = pluginport.PluginWindow
	PluginWindowHost       = pluginport.PluginWindowHost
	PluginVaultList        = pluginport.PluginVaultList
	PluginVaultListHost    = pluginport.PluginVaultListHost
	PluginVaultReader      = pluginport.PluginVaultReader
	PluginVaultStore       = pluginport.PluginVaultStore
	PluginVaultWriter      = pluginport.PluginVaultWriter
	PluginWork             = pluginport.PluginWork
)
