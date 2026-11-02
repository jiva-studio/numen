package pluginport

import "context"

// PluginCommandEntry is one command an enabled plugin declares, and the plugin
// that declares it, by its id and by its display name.
type PluginCommandEntry struct {
	PluginID   string
	PluginName string
	Command    PluginCommand
}

// PluginCommandCall is one run of a command, checked against its declaration.
type PluginCommandCall struct {
	PluginID string
	Command  string
	// Path is the file the command runs over, relative to the vault folder. Empty for a command over
	// no file.
	Path string
	// Arguments are the values by argument name, each as text.
	Arguments map[string]string
}

// PluginCommandResult is the result of a command that ran.
type PluginCommandResult struct {
	// Message is the outcome text to show to the user.
	Message string
	// OpenedPath is the file a window opens once the command ran, relative to the vault folder. Empty
	// for a command that leaves nothing to open.
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
	// RunCommand runs the command in the backend of its plugin. A plugin that cannot be started or
	// reached is an *UnavailableError, and a command that ran and failed a *CommandFailedError.
	RunCommand(ctx context.Context, call PluginCommandCall) (PluginCommandResult, error)
}
