package plugin

import "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

// CommandEntry is one command an enabled plugin declares, and the plugin
// that declares it, by its id and by its display name.
type CommandEntry struct {
	PluginID   string
	PluginName string
	Command    manifest.Command
}

// SlotEntry is one slot entry an enabled plugin declares, and the plugin that declares it.
type SlotEntry struct {
	PluginID string
	Entry    manifest.SlotEntry
}

// KeybindingEntry is one keybinding an enabled plugin declares, and the plugin that declares it.
type KeybindingEntry struct {
	PluginID   string
	Keybinding manifest.Keybinding
}

// TabKindEntry is one tab kind an enabled plugin declares, and the plugin that declares it.
type TabKindEntry struct {
	PluginID string
	TabKind  manifest.TabKind
}

// TriggerEntry is one trigger an enabled plugin declares, and the plugin that declares it.
type TriggerEntry struct {
	PluginID string
	Trigger  manifest.Trigger
}
