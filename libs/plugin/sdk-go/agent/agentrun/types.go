package agentrun

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
)

// Runner is the part of an agent that works tasks. An agent plugin implements Runner and ToolStore,
// and NewHandler serves them over the agent service.
type Runner interface {
	// Ask works the task and reports each step to send, in the order it happens, until the work is
	// done. A canceled ctx stops the work.
	Ask(ctx context.Context, req *Request, send func(*Response) error) error
	// GetState reports whether an agent can be reached.
	GetState(ctx context.Context) (*State, error)
	// GetPrograms lists the programs the plugin runs and the models each offers.
	GetPrograms(ctx context.Context, commands []ProgramCommand) ([]*Program, error)
	// FinishConversation ends the conversation and stops whatever is still answered in it.
	FinishConversation(ctx context.Context, conversation string) error
}

// ToolStore is the part of an agent that keeps what the host registers with it.
type ToolStore interface {
	// RegisterTool keeps a tool the host registers with the agent.
	RegisterTool(ctx context.Context, tool *ToolDefinition) error
	// ExtendSystemPrompt appends the extension to the system prompt and returns the whole prompt.
	ExtendSystemPrompt(ctx context.Context, extension string) (string, error)
	// GetTools returns the registered tools and the system prompt extension.
	GetTools(ctx context.Context) (*ToolSet, error)
}

// Request is a task given to an agent.
//
//numen:api
type Request struct {
	// Prompt is what the person wrote.
	Prompt string
	// Focus is the note in focus. Empty when nothing is in focus.
	Focus string
	// Conversation names the thread the task belongs to. Tasks of one name are answered in one
	// conversation. Empty is no conversation.
	Conversation string
	// Context is what the asking view shows, passed to each tool the agent calls. Each key starts with
	// the id of the plugin that wrote it and a slash.
	Context map[string]string
	// Agent is the id of the agent plugin that answers. Empty uses the installation default.
	Agent string
	// Model is the model that answers. Empty uses the default.
	Model string
	// Instructions are notified about the task.
	Instructions string
	// Tools are the names of the vault tools the agent may call.
	Tools []string
	// MaxSteps is the maximum number of steps the agent may take.
	MaxSteps int32
	// ToolDeclarations describe how the calls of the vault tools are shown.
	ToolDeclarations []toolcall.ToolDeclaration
	// Root is the folder the agent is started in. Empty starts it where the plugin runs.
	Root string
	// Command is the command line that starts the program. Empty starts the program where the
	// provider finds it.
	Command []string
	// ToolsURL is where the program reaches the vault's tools. Empty gives the program no tools.
	ToolsURL string
	// ToolsToken is what the program presents at ToolsURL.
	ToolsToken string
	// Reasoning is the level of reasoning the model works at. Empty leaves the depth to the program.
	Reasoning string
}

// Response is one step of the work.
type Response struct {
	Step Step
}

// GetStep returns the step of the response, nil for a nil response.
func (r *Response) GetStep() Step {
	if r == nil {
		return nil
	}
	return r.Step
}

// GetText returns the text of a text step, empty for any other step.
func (r *Response) GetText() string {
	if step, isText := r.GetStep().(*TextStep); isText {
		return step.Text
	}
	return ""
}

// GetStopped returns the reason of a stopped step, empty for any other step.
func (r *Response) GetStopped() string {
	if step, isStopped := r.GetStep().(*StoppedStep); isStopped {
		return step.Stopped
	}
	return ""
}

// GetToolCall returns the call of a tool call step, nil for any other step.
func (r *Response) GetToolCall() *toolcall.ToolCall {
	if step, isCall := r.GetStep().(*ToolCallStep); isCall {
		return step.ToolCall
	}
	return nil
}

// GetEditProgress returns the change of an edit progress step, nil for any other step.
func (r *Response) GetEditProgress() *EditProgress {
	if step, isEdit := r.GetStep().(*EditProgressStep); isEdit {
		return step.EditProgress
	}
	return nil
}

func (r *Response) IsModelStarted() bool {
	_, isStarted := r.GetStep().(*ModelStartedStep)
	return isStarted
}

func (r *Response) IsToolFinished() bool {
	_, isFinished := r.GetStep().(*ToolFinishedStep)
	return isFinished
}

// Step is the content of a Response: one of the step types of this package.
type Step interface {
	isStep()
}

// TextStep is a piece of what the agent is saying, appended in the order it arrives.
type TextStep struct {
	Text string
}

// ToolCallStep reports a tool the agent is using.
type ToolCallStep struct {
	ToolCall *toolcall.ToolCall
}

// StoppedStep reports that the agent has finished. Stopped is the reason, empty or "done" when the
// agent stopped because it was done.
type StoppedStep struct {
	Stopped string
}

// ToolFinishedStep reports that the tool the agent was using has finished.
type ToolFinishedStep struct{}

// ModelStartedStep reports that a request to the model has begun.
type ModelStartedStep struct{}

// EditProgressStep reports a change to a note while the call writing it is still arriving.
type EditProgressStep struct {
	EditProgress *EditProgress
}

func (*TextStep) isStep()         {}
func (*ToolCallStep) isStep()     {}
func (*StoppedStep) isStep()      {}
func (*ToolFinishedStep) isStep() {}
func (*ModelStartedStep) isStep() {}
func (*EditProgressStep) isStep() {}

// EditProgress is a change to a note, as far as the call writing it has arrived.
//
//numen:api
type EditProgress struct {
	// Call is the name the agent gave the call making the change.
	Call string
	// Path is the note the change goes into, by the path the vault files it under.
	Path string
	// Match is the text the change replaces.
	Match string
	// Text is the text that takes its place.
	Text string
}

// State is whether an agent can be reached.
//
//numen:api
type State struct {
	// Unreachable is set when no agent can be reached. It says why, in the person's own words.
	Unreachable string
}

// ProgramCommand is the command line the installation set for one program.
//
//numen:api
type ProgramCommand struct {
	// Program is the program, by the name it is listed under.
	Program string
	Command []string
}

// Program is one program a provider runs.
//
//numen:api
type Program struct {
	// Name is the name a task asks for the program by, and the settings keep its section under.
	Name string
	// Title is what the program is called when shown to a person.
	Title string
	// IsInstalled is whether the program is installed on this machine.
	IsInstalled bool
	// IsDefault is set on the program an unconfigured installation answers with.
	IsDefault bool
	// Models are the models the program offers. Empty where it offers none.
	Models []*reasoning.Model
	// LimitsTools is whether the program runs only the vault tools a task names.
	LimitsTools bool
}

// ToolDefinition describes a tool registered with the agent.
type ToolDefinition struct {
	Name                 string
	Description          string
	ParametersJSONSchema string
}

// ToolSet is the tools registered with an agent and its system prompt extension.
type ToolSet struct {
	Tools                 []*ToolDefinition
	SystemPromptExtension string
}
