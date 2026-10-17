// Package plugin manages discovery, lifecycle, and communication with external plugins.
package plugin

import (
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

const (
	// PermissionVaultRead allows reading files from the active vault.
	PermissionVaultRead = port.PermissionVaultRead

	// PermissionVaultWrite allows modifying files in the active vault.
	PermissionVaultWrite = port.PermissionVaultWrite

	// PermissionNetworkOutbound allows outbound network requests.
	PermissionNetworkOutbound = port.PermissionNetworkOutbound

	// PermissionAIComplete allows asking the host for a model completion.
	PermissionAIComplete = port.PermissionAIComplete
)

// Lifecycle defines how the host activates and evicts a plugin from memory.
type Lifecycle = port.PluginLifecycle

// Backend describes the server-side executable or wasm module of a plugin.
type Backend = port.PluginBackend

// Frontend describes the client-side bundle and styles of a plugin.
type Frontend = port.PluginFrontend

// SettingOptions is a slice of option values supporting string arrays and object arrays.
type SettingOptions = port.PluginSettingOptions

// Setting declares one configuration field managed by the core host.
type Setting = port.PluginManifestSetting

// AgentTool declares a custom tool provided or required by an agent plugin.
type AgentTool = port.PluginAgentTool

// AgentConfig declares agent system prompt extensions and custom tools in the manifest.
type AgentConfig = port.PluginAgentConfig

// Manifest is the metadata declaring a plugin's identity, capabilities, and lifecycle.
type Manifest = port.PluginManifest

// ReadManifest decodes and validates a manifest from JSON input.
func ReadManifest(r io.Reader) (Manifest, error) {
	return port.ReadManifest(r)
}
