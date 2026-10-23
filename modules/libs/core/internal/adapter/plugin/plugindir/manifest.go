package plugindir

import (
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

const (
	// PermissionVaultRead allows reading files from the active vault.
	PermissionVaultRead = pluginport.PermissionVaultRead

	// PermissionVaultWrite allows modifying files in the active vault.
	PermissionVaultWrite = pluginport.PermissionVaultWrite

	// PermissionNetworkOutbound allows outbound network requests.
	PermissionNetworkOutbound = pluginport.PermissionNetworkOutbound

	// PermissionAIComplete allows asking the host for a model completion.
	PermissionAIComplete = pluginport.PermissionAIComplete
)

// Lifecycle defines how the host activates and evicts a plugin from memory.
type Lifecycle = pluginport.PluginLifecycle

// Backend describes the server-side executable or wasm module of a plugin.
type Backend = pluginport.PluginBackend

// Frontend describes the client-side bundle and styles of a plugin.
type Frontend = pluginport.PluginFrontend

// SettingOptions is a slice of option values supporting string arrays and object arrays.
type SettingOptions = pluginport.PluginSettingOptions

// Setting declares one configuration field managed by the core host.
type Setting = pluginport.PluginManifestSetting

// Manifest is the metadata declaring a plugin's identity, capabilities, and lifecycle.
type Manifest = pluginport.PluginManifest

// ReadManifest decodes and validates a manifest from JSON input.
func ReadManifest(r io.Reader) (Manifest, error) {
	return pluginport.ReadManifest(r)
}
