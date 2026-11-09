package agentport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Agent takes tasks. It works a vault on behalf of the user through tools. The
// agent is an external program reached over its own protocol.
//
// Callers see one agent and never which program serves it.
type Agent interface {
	// StartRun gives the agent a task and returns the run it has begun.
	StartRun(ctx context.Context, task Task) (Run, error)
	// EndConversation ends a conversation. The agent releases its state for the
	// conversation and stops and waits for any work still running in it.
	// An empty conversation is a no-op.
	EndConversation(ctx context.Context, conversation string) error
}

// Task is the user's question, the focus it originates from, and the
// conversation it belongs to.
type Task struct {
	Question         string
	Focus            string
	Conversation     string
	Agent            string
	Model            string
	Reasoning        string
	Instructions     string
	Tools            []string
	MaxSteps         int
	Root             string
	Command          []string
	Endpoint         ToolEndpoint
	Context          map[string]string
	Vault            domain.VaultID
	ToolDeclarations []ToolDeclaration
}

// ToolEndpoint is the address of the vault's tools and the credential presented there.
type ToolEndpoint struct {
	URL   string
	Token string
}

// read while it is being written.
type ToolDeclaration struct {
	Name    string
	Title   string
	Kind    domain.StepKind
	About   string
	Element string
	Match   string
	Text    string
}

// Run is one task being worked. Task is the request; this is its execution.
type Run interface {
	// GetSteps returns the steps in the order the agent takes them. The channel
	// closes when there are no more.
	GetSteps() <-chan Step
	// Stop ends the run and returns once nothing of it is still running.
	Stop() error
}

// Step is one event of a run: agent text, a tool call, or a stop.
type Step struct {
	Kind   domain.StepKind
	Call   string
	Text   string
	Tool   string
	About  string
	Place  domain.Place
	Count  int
	Match  string
	Detail string
}

// ReasoningLevel is one reasoning depth a model supports.
type ReasoningLevel = domain.ReasoningLevel

// AgentModel is one model a program offers.
type AgentModel struct {
	Name             string
	Title            string
	IsDefault        bool
	ReasoningLevels  []ReasoningLevel
	DefaultReasoning string
}

// AgentProgram is one program a provider runs.
type AgentProgram struct {
	Provider    string
	Name        string
	Title       string
	IsInstalled bool
	IsDefault   bool
	Models      []AgentModel
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

// AgentProvider runs tasks on the programs that plugins provide and lists those programs.
type AgentProvider interface {
	Agent
	// GetPrograms lists the programs the providers run, with the models each offers. Commands are the
	// command lines the installation set, by program. A provider that cannot be reached lists none,
	// and the error reports it alongside the programs of the other providers.
	GetPrograms(ctx context.Context, commands map[string][]string) ([]AgentProgram, error)
}

// CompletionPrograms finds the program a route names and starts the tasks that
// carry the questions to it.
type CompletionPrograms interface {
	// GetProgram returns the program named, or, when the name is empty, the one
	// the settings name, else the default of the plugins. A program that is not
	// installed or that no plugin lists fails with completion.ErrNoRoute, and plugins that
	// cannot be reached with ErrUnavailable.
	GetProgram(ctx context.Context, name string) (AgentProgram, error)
	// StartRun starts the task on the program it names, under that program's
	// section of the settings.
	StartRun(ctx context.Context, task Task) (Run, error)
}

// stoppedDone is the reason an agent gives when it stopped because it was done.
const stoppedDone = "done"

var errNoAnswer = errors.New("agent returned empty response")

// Starter starts the tasks an agent is given.
type Starter interface {
	StartRun(ctx context.Context, task Task) (Run, error)
}

// Answer gives the agent one question and returns its reply. The task is sent with no tools, no
// tool endpoint and no conversation, so the agent has no access to the vault. An agent that
// cannot be started, stops with an error or returns no text fails with
// domain.ErrUnavailable. Cancellation of the context stops the run and fails with the context's error.
func Answer(ctx context.Context, agent Starter, task Task) (_ string, err error) {
	task.Tools = nil
	task.ToolDeclarations = nil
	task.Endpoint = ToolEndpoint{}
	task.Conversation = ""

	run, err := agent.StartRun(ctx, task)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", fmt.Errorf("%w: %w", domain.ErrUnavailable, err)
	}
	defer func() { err = errors.Join(err, run.Stop()) }()
	return readAnswer(ctx, run.GetSteps())
}

func readAnswer(ctx context.Context, steps <-chan Step) (string, error) {
	var response strings.Builder
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case step, isOpen := <-steps:
			if !isOpen {
				return finishAnswer(ctx, response.String())
			}
			if step.Kind == domain.StepSay {
				response.WriteString(step.Text)
			}
			if step.Kind == domain.StepStop && step.Detail != "" && step.Detail != stoppedDone {
				return "", failAnswer(ctx, fmt.Errorf("the agent stopped: %s", step.Detail))
			}
		}
	}
}

func finishAnswer(ctx context.Context, response string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(response) == "" {
		return "", fmt.Errorf("%w: %w", domain.ErrUnavailable, errNoAnswer)
	}
	return response, nil
}

func failAnswer(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return fmt.Errorf("%w: %w", domain.ErrUnavailable, err)
}
