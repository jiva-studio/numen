package pluginport

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// PluginCommandEntry is one command an enabled plugin declares, and the plugin
// that declares it, by its id and by its display name.
type PluginCommandEntry struct {
	PluginID   string
	PluginName string
	Command    PluginCommand
}

// PluginSlotEntry is one slot entry an enabled plugin declares, and the plugin that declares it.
type PluginSlotEntry struct {
	PluginID string
	Entry    manifest.SlotEntry
}

// PluginParts is what one plugin has and whether the host runs it.
type PluginParts struct {
	HasBackend  bool
	HasFrontend bool
	IsSuspended bool
}

// PluginCommandCall is one run of a command, checked against its declaration.
type PluginCommandCall struct {
	PluginID  string
	Command   string
	Path      string
	Arguments map[string]string
}

// PluginCommandResult is the result of a command that ran.
type PluginCommandResult struct {
	Message    string
	OpenedPath string
}

// CommandFailedError is a command the plugin ran and that failed, with the plugin's message.
type CommandFailedError struct {
	Message string
}

// Error returns the plugin's message.
func (e *CommandFailedError) Error() string {
	return e.Message
}

// PluginCommands is the commands the plugins declare and the backends that run them.
type PluginCommands interface {
	// ListCommands lists the commands of every enabled plugin, in the order of the plugins' ids and
	// then of each manifest.
	ListCommands() []PluginCommandEntry
	// ListSlots lists the slot entries of every enabled plugin, in the order of the plugins' ids and
	// then of each manifest.
	ListSlots() []PluginSlotEntry
	// ListHandlers lists the handlers of shared commands of every enabled plugin, in the order of the
	// plugins' ids and then of each manifest, each as a command under the full id of its shared
	// command, with the handler's title and condition.
	ListHandlers() []PluginCommandEntry
	// GetParts says what the plugin of the id has and whether it is suspended. A plugin that is not
	// enabled has none.
	GetParts(pluginID string) PluginParts
	// RunCommand runs the command in the backend of its plugin. A plugin that cannot be started or
	// reached is an *UnavailableError, and a command that ran and failed a *CommandFailedError.
	RunCommand(ctx context.Context, call PluginCommandCall) (PluginCommandResult, error)
}
