package pluginport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// PluginDeclarations is what the manifests of the enabled plugins declare, whether or not a plugin is
// running.
type PluginDeclarations interface {
	// ListCommands lists the commands of every enabled plugin, in the order of the plugins' ids and
	// then of each manifest.
	ListCommands() []plugin.CommandEntry
	// ListSlots lists the slot entries of every enabled plugin, in the order of the plugins' ids and
	// then of each manifest.
	ListSlots() []plugin.SlotEntry
	// ListKeybindings lists the keybindings of every enabled plugin, in the order of the plugins' ids and
	// then of each manifest.
	ListKeybindings() []plugin.KeybindingEntry
	// ListHandlers lists the handlers of shared commands of every enabled plugin, in the order of the
	// plugins' ids and then of each manifest, each as a command under the full id of its shared
	// command, with the handler's title and condition.
	ListHandlers() []plugin.CommandEntry
}

// PluginCommands is the commands the plugins declare and the backends that run them.
type PluginCommands interface {
	PluginDeclarations
	// GetParts says what the plugin of the id has and whether it is suspended. A plugin that is not
	// enabled has none.
	GetParts(pluginID string) plugin.Parts
	// RunCommand runs the command in the backend of its plugin. A plugin that cannot be started or
	// reached is an *UnavailableError, and a command that ran and failed a *CommandFailedError.
	RunCommand(ctx context.Context, call plugin.CommandCall) (plugin.CommandResult, error)
}
