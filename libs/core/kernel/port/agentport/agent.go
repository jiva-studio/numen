package agentport

import (
	"context"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
)

// Agent takes tasks. It works a vault on behalf of the user through tools. The
// agent is an external program reached over its own protocol.
//
// Callers see one agent and never which program serves it.
type Agent interface {
	// StartRun gives the agent a task and returns the run it has begun.
	StartRun(ctx context.Context, task agentdomain.Task) (Run, error)
	// EndConversation ends a conversation. The agent releases its state for the
	// conversation and stops and waits for any work still running in it.
	// An empty conversation is a no-op.
	EndConversation(ctx context.Context, conversation string) error
}

// Run is one task being worked. Task is the request; this is its execution.
type Run interface {
	// GetSteps returns the steps in the order the agent takes them. The channel
	// closes when there are no more.
	GetSteps() <-chan agentdomain.Step
	// Stop ends the run and returns once nothing of it is still running.
	Stop() error
}

// AgentProvider runs tasks on the programs that plugins provide and lists those programs.
type AgentProvider interface {
	Agent
	// GetPrograms lists the programs the providers run, with the models each offers. Commands are the
	// command lines the installation set, by program. A provider that cannot be reached lists none,
	// and the error reports it alongside the programs of the other providers.
	GetPrograms(ctx context.Context, commands map[string][]string) ([]agentdomain.Program, error)
}
