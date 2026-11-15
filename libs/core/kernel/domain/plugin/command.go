package plugin

// Parts is what one plugin has and whether the host runs it.
type Parts struct {
	HasBackend  bool
	HasFrontend bool
	IsSuspended bool
	Name        string
	TextKinds   map[string]string
}

// CommandCall is one run of a command, checked against its declaration. Paths, set by a trigger on
// note.frontmatterChanged in place of Path, are the notes the one run covers.
type CommandCall struct {
	PluginID  string
	Command   string
	Path      string
	Paths     []string
	Arguments map[string]string
	ItemID    string
}

type CommandResult struct {
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
