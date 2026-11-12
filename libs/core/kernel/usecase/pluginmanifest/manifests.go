// Package pluginmanifest lists what the manifests of the enabled plugins declare: commands, handlers,
// triggers, slot entries and tab kinds, and the parts of one plugin. It starts no plugin.
package pluginmanifest

import (
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Host provides the plugins, their manifests and whether the host holds one suspended.
type Host interface {
	GetPluginInfos() []plugindomain.Info
	GetPlugin(id string) (manifest.Discovered, bool)
	IsPluginSuspended(pluginID string) bool
}

// Manifests is the manifests of the plugins the host holds.
type Manifests struct {
	host Host
}

// New lists over the plugins of the host.
func New(host Host) Manifests {
	return Manifests{host: host}
}

// ListCommands lists the commands the manifests of the enabled plugins declare. No plugin is started.
func (m Manifests) ListCommands() []plugindomain.CommandEntry {
	var found []plugindomain.CommandEntry
	for _, info := range m.host.GetPluginInfos() {
		plugin, isEnabled := m.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, command := range plugin.Manifest.Contributes.Commands {
			found = append(found, plugindomain.CommandEntry{
				PluginID: plugin.Manifest.ID, PluginName: plugin.Manifest.Name, Command: command,
			})
		}
	}
	return found
}

// ListTabKinds lists the tab kinds the manifests of the enabled plugins declare. A plugin that declares a
// tab kind, a reader or a type of dropped data an earlier plugin already declares is left out whole.
// No plugin is started.
func (m Manifests) ListTabKinds() []plugindomain.TabKindEntry {
	var found []plugindomain.TabKindEntry
	for _, plugin := range m.listUncontestedPlugins() {
		for _, kind := range plugin.Manifest.Contributes.TabKinds {
			found = append(found, plugindomain.TabKindEntry{PluginID: plugin.Manifest.ID, TabKind: kind})
		}
	}
	return found
}

func (m Manifests) listUncontestedPlugins() []manifest.Discovered {
	var enabled []manifest.Discovered
	for _, info := range m.host.GetPluginInfos() {
		if plugin, isEnabled := m.host.GetPlugin(info.ID); isEnabled {
			enabled = append(enabled, plugin)
		}
	}
	conflicting := map[string]bool{}
	for _, conflict := range manifest.ListContributionConflicts(enabled) {
		conflicting[conflict.PluginID] = true
	}
	return slices.DeleteFunc(enabled, func(plugin manifest.Discovered) bool { return conflicting[plugin.Manifest.ID] })
}

// ListSlots lists the slot entries the manifests of the enabled plugins declare. No plugin is started.
func (m Manifests) ListSlots() []plugindomain.SlotEntry {
	var found []plugindomain.SlotEntry
	for _, info := range m.host.GetPluginInfos() {
		plugin, isEnabled := m.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, entry := range plugin.Manifest.Contributes.Slots {
			found = append(found, plugindomain.SlotEntry{PluginID: plugin.Manifest.ID, Entry: entry})
		}
	}
	return found
}

// ListKeybindings lists the keybindings the manifests of the enabled plugins declare. No plugin is started.
func (m Manifests) ListKeybindings() []plugindomain.KeybindingEntry {
	var found []plugindomain.KeybindingEntry
	for _, info := range m.host.GetPluginInfos() {
		plugin, isEnabled := m.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, entry := range plugin.Manifest.Contributes.Keybindings {
			found = append(found, plugindomain.KeybindingEntry{PluginID: plugin.Manifest.ID, Keybinding: entry})
		}
	}
	return found
}

// ListHandlers lists the handlers the manifests of the enabled plugins declare, each as a command under
// the full id of its shared command. No plugin is started.
func (m Manifests) ListHandlers() []plugindomain.CommandEntry {
	var found []plugindomain.CommandEntry
	for _, info := range m.host.GetPluginInfos() {
		plugin, isEnabled := m.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, handler := range plugin.Manifest.Contributes.Handlers {
			found = append(found, plugindomain.CommandEntry{
				PluginID: plugin.Manifest.ID, PluginName: plugin.Manifest.Name,
				Command: manifest.Command{
					ID: handler.Command, Title: handler.Title, When: handler.When,
					Arguments: manifest.GetSharedCommandArguments(handler.Command),
				},
			})
		}
	}
	return found
}

// ListTriggers lists the triggers the manifests of the enabled plugins declare, leaving out those of a
// plugin the host holds suspended. No plugin is started.
func (m Manifests) ListTriggers() []plugindomain.TriggerEntry {
	var found []plugindomain.TriggerEntry
	for _, info := range m.host.GetPluginInfos() {
		plugin, isEnabled := m.host.GetPlugin(info.ID)
		if !isEnabled || m.host.IsPluginSuspended(info.ID) {
			continue
		}
		for _, trigger := range plugin.Manifest.Contributes.Triggers {
			found = append(found, plugindomain.TriggerEntry{PluginID: plugin.Manifest.ID, Trigger: trigger})
		}
	}
	return found
}

// GetParts says what the enabled plugin of the id has, and whether the host holds it suspended.
func (m Manifests) GetParts(pluginID string) plugindomain.Parts {
	plugin, isEnabled := m.host.GetPlugin(pluginID)
	if !isEnabled {
		return plugindomain.Parts{}
	}
	return plugindomain.Parts{
		HasBackend: plugin.Manifest.HasBackend(), HasFrontend: plugin.Manifest.HasFrontend(),
		IsSuspended: m.host.IsPluginSuspended(pluginID),
		Name:        plugin.Manifest.Name, TextKinds: listTextKinds(plugin.Manifest),
	}
}

func listTextKinds(m manifest.Manifest) map[string]string {
	if len(m.Contributes.TextKinds) == 0 {
		return nil
	}
	titles := make(map[string]string, len(m.Contributes.TextKinds))
	for _, one := range m.Contributes.TextKinds {
		titles[one.Kind] = one.Title
	}
	return titles
}

// ListFrontendPluginIDs lists the ids of the plugins that have a frontend, in the order of the host.
func (m Manifests) ListFrontendPluginIDs() []string {
	infos := m.host.GetPluginInfos()
	ids := make([]string, 0, len(infos))
	for _, info := range infos {
		if info.HasFrontend {
			ids = append(ids, info.ID)
		}
	}
	return ids
}
