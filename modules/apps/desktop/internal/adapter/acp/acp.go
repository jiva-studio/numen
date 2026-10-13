// Package acp reaches external coding agents communicating over JSON streams.
package acp

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/runner"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
)

// Endpoint is where the agent reaches the tools of this vault over MCP.
type Endpoint struct {
	URL   string
	Token string
}

// Options is what an ACP agent is made from.
type Options struct {
	// Program names the agent: agent.UseAntigravity or agent.UseCodex.
	Program string
	// Settings is how the installation runs it.
	Settings agent.AgentProgram
	// Root is the folder the agent is started in.
	Root string
	// Tools is where it reaches this vault over MCP.
	Tools Endpoint
	// ErrorHandler is notified of background execution errors.
	ErrorHandler func(error)
}

// New makes an ACP agent, waiting to be asked something.
func New(opts Options) *Agent {
	return &Agent{
		Program:      opts.Program,
		Command:      opts.Settings.Command,
		Root:         opts.Root,
		Tools:        opts.Tools,
		Model:        opts.Settings.Model,
		ErrorHandler: opts.ErrorHandler,
	}
}

// Agent reaches an external agent CLI using standard Agent Client Protocol (ACP).
type Agent struct {
	// Program names the agent identifier ("antigravity", "codex", etc.)
	Program string
	// Command starts it. Empty uses the discovered binary.
	Command []string
	// Root is the folder the agent is started in.
	Root string
	// Tools is where it reaches this vault over MCP.
	Tools Endpoint
	// Model is which model answers.
	Model string
	// ErrorHandler is notified of background execution errors.
	ErrorHandler func(error)

	// registry holds the work this agent started and the session each conversation is on.
	registry runner.Registry
}

// Close stops every running agent process.
func (a *Agent) Close() error { return a.registry.Close() }

// EndConversation stops everything still being answered in the conversation and lets go of its
// session.
func (a *Agent) EndConversation(ctx context.Context, conversation string) error {
	return a.registry.EndConversation(ctx, conversation)
}

func (a *Agent) resolveCommand() (string, []string) {
	if len(a.Command) > 0 {
		return a.Command[0], a.Command[1:]
	}
	if a.Program == agent.UseCodex {
		return agent.FindCodex(), nil
	}
	return agent.FindAntigravity(), nil
}
