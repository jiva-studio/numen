package port

import "context"

// ToolEffect is what a tool a plugin offers does to the vault.
type ToolEffect int

const (
	// ToolReads is a tool that reads and changes nothing.
	ToolReads ToolEffect = iota + 1
	// ToolWrites is a tool that changes the vault.
	ToolWrites
)

// ToolServer is one of the servers an agent is given tools on.
type ToolServer int

const (
	// ToolServerFull is the server an agent works the whole vault through.
	ToolServerFull ToolServer = iota + 1
	// ToolServerReading is the server an agent answers from, whose every tool reads.
	ToolServerReading
	// ToolServerReview is the server of the window a person reviews cards in.
	ToolServerReview
)

// ToolStep is how a call of a tool is shown to a person.
type ToolStep int

const (
	// ToolStepByEffect leaves it to the effect: a look for a tool that reads, an edit for one that writes.
	ToolStepByEffect ToolStep = iota
	// ToolStepSearch is a call that looks for notes.
	ToolStepSearch
	// ToolStepRemove is a call that takes something out of the vault.
	ToolStepRemove
	// ToolStepMove is a call that files something elsewhere or under another name.
	ToolStepMove
)

// PluginTool is one tool a plugin offers an agent, and the plugin that does.
type PluginTool struct {
	PluginID string
	Name     string
	Title    string
	// Description is what the tool is for and when an agent reaches for it.
	Description string
	// InputSchema is the JSON schema of the arguments, as a JSON object.
	InputSchema []byte
	// OutputSchema is the JSON schema of the structured result. Empty where the tool answers
	// with text alone.
	OutputSchema []byte
	Effect       ToolEffect
	// Step is how a call is shown to a person.
	Step ToolStep
	// Servers are the servers the tool is offered on. None is the full server alone.
	Servers []ToolServer
}

// IsOn says whether the tool is offered on the server.
func (t PluginTool) IsOn(server ToolServer) bool {
	if len(t.Servers) == 0 {
		return server == ToolServerFull
	}
	for _, one := range t.Servers {
		if one == server {
			return true
		}
	}
	return false
}

// PluginToolResult is what a tool answered.
type PluginToolResult struct {
	Text string
	// Structured is the result as JSON. Empty where the tool listed no output schema.
	Structured []byte
	IsRefusal bool
}

// PluginTools is what the plugins offer an agent as tools. Where no plugin offers any, nothing
// is listed.
type PluginTools interface {
	// ListTools lists the tools of every plugin that offers any and holds the permission its
	// effect needs. A plugin that cannot be started or reached is left out.
	ListTools(ctx context.Context) []PluginTool
	// CallTool runs the tool the plugin listed. A plugin that no longer holds the permission, or
	// cannot be reached, is an *UnavailableError or ErrPermissionDenied.
	CallTool(ctx context.Context, pluginID, name string, arguments []byte, toolContext map[string]string) (PluginToolResult, error)
}
