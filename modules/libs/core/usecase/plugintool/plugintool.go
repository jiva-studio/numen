package plugintool

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The ports' words for a tool, which a driving adapter reaches here.
type (
	// Tool is one tool a plugin offers an agent.
	Tool = port.PluginTool
	// Result is what a tool answered.
	Result = port.PluginToolResult
	// Effect is what a tool does to the vault.
	Effect = port.ToolEffect
	// Step is how a call of a tool is shown to a person.
	Step = port.ToolStep
	// Server is one of the servers an agent is given tools on.
	Server = port.ToolServer
)

const (
	// EffectReads is a tool that reads and changes nothing.
	EffectReads = port.ToolReads
	// EffectWrites is a tool that changes the vault.
	EffectWrites = port.ToolWrites

	// StepByEffect leaves it to the effect how a call is shown.
	StepByEffect = port.ToolStepByEffect
	// StepSearch is a call that looks for notes.
	StepSearch = port.ToolStepSearch
	// StepRemove is a call that takes something out of the vault.
	StepRemove = port.ToolStepRemove
	// StepMove is a call that files something elsewhere or under another name.
	StepMove = port.ToolStepMove

	// ServerFull is the server an agent works the whole vault through.
	ServerFull = port.ToolServerFull
	// ServerReading is the server an agent answers from, whose every tool reads.
	ServerReading = port.ToolServerReading
	// ServerReview is the server of the window a person reviews cards in.
	ServerReview = port.ToolServerReview
)

// Tools is the scenarios that offer an agent the tools the plugins serve.
type Tools struct {
	List List
	Call Call
}

// NewTools makes both scenarios over the tools the plugins offer, which may be nothing.
func NewTools(tools port.PluginTools) Tools {
	return Tools{List: NewList(tools), Call: NewCall(tools)}
}

// List lists the tools the plugins offer an agent.
type List struct {
	tools port.PluginTools
}

// NewList makes the scenario over the tools the plugins offer.
func NewList(tools port.PluginTools) List {
	return List{tools: tools}
}

// Execute lists the tools of every plugin that offers any and holds the permission each needs.
// Where there is no plugin to ask it lists none.
func (u List) Execute(ctx context.Context) []port.PluginTool {
	if u.tools == nil {
		return nil
	}
	return u.tools.ListTools(ctx)
}

// Call runs one tool a plugin offers.
type Call struct {
	tools port.PluginTools
}

// NewCall makes the scenario over the tools the plugins offer.
func NewCall(tools port.PluginTools) Call {
	return Call{tools: tools}
}

// Execute runs the tool the plugin listed with the arguments, a JSON object, and what the person
// is looking at in the window the agent was started from.
func (u Call) Execute(
	ctx context.Context, pluginID, name string, arguments []byte, toolContext map[string]string,
) (port.PluginToolResult, error) {
	if u.tools == nil {
		return port.PluginToolResult{}, port.ErrUnavailable
	}
	return u.tools.CallTool(ctx, pluginID, name, arguments, toolContext)
}
