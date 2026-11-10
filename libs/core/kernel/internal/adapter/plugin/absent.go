package plugin

import (
	"context"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Absent stands for the plugin host where the installation has none, or where plugins are off: no
// plugin is enabled, none declares a secret and no manifest declares anything.
type Absent struct{}

// IsEnabled reports that no plugin is enabled.
func (Absent) IsEnabled(string) bool { return false }

// ListDeclarations returns no declaration.
func (Absent) ListDeclarations() []plugindomain.DeclarationKey { return nil }

// Refresh configures no plugin.
func (Absent) Refresh(context.Context, ...string) error { return nil }

// ListCommands returns no command.
func (Absent) ListCommands() []plugindomain.CommandEntry { return nil }

// ListSlots returns no slot entry.
func (Absent) ListSlots() []plugindomain.SlotEntry { return nil }

// ListKeybindings returns no keybinding.
func (Absent) ListKeybindings() []plugindomain.KeybindingEntry { return nil }

// ListHandlers returns no handler.
func (Absent) ListHandlers() []plugindomain.CommandEntry { return nil }

// GetParts says the plugin has nothing.
func (Absent) GetParts(string) plugindomain.Parts { return plugindomain.Parts{} }
