package container

import (
	"context"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// absentPlugins stands for the plugin host where the installation has none, or where plugins are off:
// nothing is enabled, no plugin declares a secret and no manifest declares anything.
type absentPlugins struct{}

func (absentPlugins) IsEnabled(string) bool { return false }

func (absentPlugins) ListDeclarations() []plugindomain.DeclarationKey { return nil }

func (absentPlugins) Refresh(context.Context, ...string) error { return nil }

func (absentPlugins) ListCommands() []plugindomain.CommandEntry { return nil }

func (absentPlugins) ListSlots() []plugindomain.SlotEntry { return nil }

func (absentPlugins) ListKeybindings() []plugindomain.KeybindingEntry { return nil }

func (absentPlugins) ListHandlers() []plugindomain.CommandEntry { return nil }

func (absentPlugins) GetParts(string) plugindomain.Parts { return plugindomain.Parts{} }
