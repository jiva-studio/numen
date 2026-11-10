package agent

import (
	"context"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

// Starter starts the tasks an agent is given.
type Starter = agentport.Starter

// Answer sends the agent one question and returns its reply. The task is sent
// with no tool, no tool endpoint and no conversation, so the agent answers on
// its own and has no access to the vault. An agent that cannot be started, stops
// for its own reason or returns nothing fails with port.ErrUnavailable. A
// context that ends stops the run and fails with the context's error.
func Answer(ctx context.Context, agent Starter, task agentdomain.Task) (string, error) {
	return agentport.Answer(ctx, agent, task)
}
