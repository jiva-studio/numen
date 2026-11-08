// Package manifestlist lists what the manifests of the enabled plugins declare: commands, handlers,
// triggers, slot entries and tab kinds, and the parts of one plugin. It starts no plugin.
package manifestlist

import (
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// Host provides the plugins, their manifests and whether the host holds one suspended.
type Host interface {
	pluginport.PluginDiscoverer
	pluginport.PluginResolver
}

// Manifests is the manifests of the plugins the host holds.
type Manifests struct {
	host Host
}

// Open lists over the plugins of the host.
func Open(host Host) Manifests {
	return Manifests{host: host}
}

// ListCommands lists the commands the manifests of the enabled plugins declare. No plugin is started.
func (m Manifests) ListCommands() []pluginport.PluginCommandEntry {
	var found []pluginport.PluginCommandEntry
	for _, info := range m.host.GetPluginInfos() {
		plugin, isEnabled := m.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, command := range plugin.Manifest.Contributes.Commands {
			found = append(found, pluginport.PluginCommandEntry{
				PluginID: plugin.Manifest.ID, PluginName: plugin.Manifest.Name, Command: command,
			})
		}
	}
	return found
}

// ListTabKinds lists the tab kinds the manifests of the enabled plugins declare. No plugin is started.
func (m Manifests) ListTabKinds() []pluginport.PluginTabKindEntry {
	var found []pluginport.PluginTabKindEntry
	for _, info := range m.host.GetPluginInfos() {
		plugin, isEnabled := m.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, kind := range plugin.Manifest.Contributes.TabKinds {
			found = append(found, pluginport.PluginTabKindEntry{PluginID: plugin.Manifest.ID, TabKind: kind})
		}
	}
	return found
}

// ListSlots lists the slot entries the manifests of the enabled plugins declare. No plugin is started.
func (m Manifests) ListSlots() []pluginport.PluginSlotEntry {
	var found []pluginport.PluginSlotEntry
	for _, info := range m.host.GetPluginInfos() {
		plugin, isEnabled := m.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, entry := range plugin.Manifest.Contributes.Slots {
			found = append(found, pluginport.PluginSlotEntry{PluginID: plugin.Manifest.ID, Entry: entry})
		}
	}
	return found
}

// ListHandlers lists the handlers the manifests of the enabled plugins declare, each as a command under
// the full id of its shared command. No plugin is started.
func (m Manifests) ListHandlers() []pluginport.PluginCommandEntry {
	var found []pluginport.PluginCommandEntry
	for _, info := range m.host.GetPluginInfos() {
		plugin, isEnabled := m.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, handler := range plugin.Manifest.Contributes.Handlers {
			found = append(found, pluginport.PluginCommandEntry{
				PluginID: plugin.Manifest.ID, PluginName: plugin.Manifest.Name,
				Command: pluginport.PluginCommand{
					ID: handler.Command, Title: handler.Title, When: handler.When,
					Arguments: pluginport.GetSharedCommandArguments(handler.Command),
				},
			})
		}
	}
	return found
}

// ListTriggers lists the triggers the manifests of the enabled plugins declare, leaving out those of a
// plugin the host holds suspended. No plugin is started.
func (m Manifests) ListTriggers() []pluginport.PluginTriggerEntry {
	var found []pluginport.PluginTriggerEntry
	for _, info := range m.host.GetPluginInfos() {
		plugin, isEnabled := m.host.GetPlugin(info.ID)
		if !isEnabled || info.Failure != nil {
			continue
		}
		for _, trigger := range plugin.Manifest.Contributes.Triggers {
			found = append(found, pluginport.PluginTriggerEntry{PluginID: plugin.Manifest.ID, Trigger: trigger})
		}
	}
	return found
}

// GetParts says what the enabled plugin of the id has, and whether the host holds it suspended.
func (m Manifests) GetParts(pluginID string) pluginport.PluginParts {
	plugin, isEnabled := m.host.GetPlugin(pluginID)
	if !isEnabled {
		return pluginport.PluginParts{}
	}
	return pluginport.PluginParts{
		HasBackend: plugin.Manifest.HasBackend(), HasFrontend: plugin.Manifest.HasFrontend(),
		IsSuspended: m.host.IsPluginSuspended(pluginID),
	}
}

// NoManifests is an installation with no plugin.
type NoManifests struct{}

func (NoManifests) ListCommands() []pluginport.PluginCommandEntry { return nil }

func (NoManifests) ListSlots() []pluginport.PluginSlotEntry { return nil }

func (NoManifests) ListHandlers() []pluginport.PluginCommandEntry { return nil }

func (NoManifests) GetParts(string) pluginport.PluginParts { return pluginport.PluginParts{} }
