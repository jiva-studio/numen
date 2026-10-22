package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Agent takes tasks. It works a vault on a person's behalf, through tools: the
// window asks, and something outside this application answers.
//
// What answers is a program of somebody else's making, reached over a protocol
// of its own. Everything above this interface sees one agent and never which.
type Agent interface {
	// StartRun gives the agent a task and returns the run it has begun.
	StartRun(ctx context.Context, task Task) (Run, error)
	// EndConversation says a conversation is over. What the agent kept of it is let go
	// of, and whatever is still being worked in it is stopped and waited for.
	// Empty is no conversation, and there is nothing to finish.
	EndConversation(ctx context.Context, conversation string) error
}

// Task is what the person asked, where they were looking when they asked it,
// and which conversation they asked it in.
type Task struct {
	Question string
	Focus    string
	// Conversation is which conversation thread this question belongs to, named by
	// the caller. Questions carrying one name are answered in one conversation
	// with the agent, so the name has to be the caller's alone: unique among
	// the conversations it has open, and never given to a second one for as
	// long as this process runs. Empty is no conversation, and a question
	// asked under it is answered on its own.
	Conversation string
	// Agent is which agent answers for this task (e.g. "antigravity", "codex", "claude").
	// Empty falls back to the configured default agent.
	Agent string
	// Model is which model answers for this task. Empty uses default.
	Model string
	// Reasoning is the depth of reasoning the model works at, by the name of one of the levels it offers.
	// Empty leaves the depth to the program.
	Reasoning string
	// Instructions is what the agent is told about the person it answers.
	Instructions string
	// Tools are the vault tools the agent may call, by the name the vault serves them under.
	// Nothing outside them is allowed.
	Tools []string
	// MaxSteps is how many steps the agent may take before it is stopped. Zero leaves the limit
	// to the agent.
	MaxSteps int
	// Root is the folder the agent works in. Empty leaves it where the agent runs.
	Root string
	// Command is the command line that starts the program the task names, as the installation set it.
	// Empty starts the program where the provider finds it.
	Command []string
	// Options are the settings only the program knows, by key, each as the JSON the installation holds.
	Options map[string]string
	// Endpoint is where the agent reaches the vault's tools. Empty gives it none.
	Endpoint ToolEndpoint
	// written.
	ToolDeclarations []ToolDeclaration
}

// ToolEndpoint is where an agent reaches the vault's tools, and what it presents there.
type ToolEndpoint struct {
	URL   string
	Token string
}

// read while it is being written.
type ToolDeclaration struct {
	// Name is the tool, by the name the vault serves it under.
	Name string
	// Title is what the tool is called when shown to a person.
	Title string
	// Kind is what a call of the tool does to the vault. A tool that declares nothing about it is
	// domain.StepToolCall.
	Kind domain.StepKind
	// About names the argument that says what a call is about.
	About string
	// Element names the field of one element that says which element it is, for a call that takes a
	// collection.
	Element string
	// Match and Text name the arguments carrying the text a call replaces and the text that takes its
	// place. Both are empty for a call that replaces no text.
	Match string
	Text  string
}

// Run is one task being worked. Task is the request; this is its execution.
type Run interface {
	// GetSteps returns the steps in the order the agent takes them. The channel
	// closes when there are no more.
	GetSteps() <-chan Step
	// Stop ends the run and returns once nothing of it is still running.
	Stop() error
}

// Step is one thing the agent said, did, or stopped for.
type Step struct {
	Kind domain.StepKind
	// Call is what the agent named this call. Every step of one call carries the
	// same name, and a step that is not a call carries none.
	Call string
	// Text is what the agent is saying, for a step that says something.
	Text string
	// Tool is what it is using, by the name it is served under.
	Tool string
	// About is what that call is about — a note, a query — empty when the
	// call says nothing worth showing.
	About string
	// Place is where in the vault the call is working, empty when what the
	// call is about is not a source the vault holds.
	Place domain.Place
	// Count is how much of the call has been written, in characters. A call
	// carrying the text of a note is written for minutes, and this is the only
	// thing that moves while it is.
	Count int
	// Match is the text a change replaces, for a step that draws one.
	Match string
	// Detail is why the work stopped, empty when the agent was done.
	Detail string
}

// ReasoningLevel is one depth of reasoning a model can be asked to work at.
type ReasoningLevel struct {
	// Name is what a task asks for the level by.
	Name string
	// Title is what the level is called when shown to a person.
	Title string
}

// AgentModel is one model a program offers.
type AgentModel struct {
	Name  string
	Title string
	// IsDefault is set where the program names the model it answers with unasked.
	IsDefault bool
	// ReasoningLevels are the depths of reasoning the model can be asked to work at, from the least to
	// the most. None where it offers no choice.
	ReasoningLevels []ReasoningLevel
	// DefaultReasoning is the level the model works at unasked, empty where it names none.
	DefaultReasoning string
}

// AgentProgram is one program a provider runs.
type AgentProgram struct {
	// Provider is the plugin that runs it, by its identifier.
	Provider string
	// Name is what a task asks for the program by, and what the settings keep its section under.
	Name string
	// Title is what the program is called when shown to a person.
	Title string
	// IsInstalled is set where the program is on this machine.
	IsInstalled bool
	// IsDefault is set on the program an installation nobody has configured answers with.
	IsDefault bool
	// Models are the models the program says it offers, none where it says none.
	Models []AgentModel
	// LimitsTools is set where the program runs only the vault tools a task names.
	LimitsTools bool
}

// GetDefaultProgram is the program an installation that names none answers with: the installed one
// marked so, else the first installed, else the one marked so, else the first.
func GetDefaultProgram(programs []AgentProgram) (AgentProgram, bool) {
	for _, installedOnly := range []bool{true, false} {
		for _, one := range programs {
			if one.IsDefault && (one.IsInstalled || !installedOnly) {
				return one, true
			}
		}
		for _, one := range programs {
			if one.IsInstalled || !installedOnly {
				return one, true
			}
		}
	}
	return AgentProgram{}, false
}

// AgentProvider runs tasks on the programs that plugins provide, and says which programs those are.
type AgentProvider interface {
	Agent
	// GetPrograms lists the programs the providers run, with the models each offers. Commands are the
	// command lines the installation set, by program. A provider that cannot be reached lists none,
	// and the error says so beside the programs of the others.
	GetPrograms(ctx context.Context, commands map[string][]string) ([]AgentProgram, error)
}
