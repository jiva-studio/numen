package agent

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// Starter starts the tasks an agent is given.
type Starter = agentport.Starter

// Answer gives the agent one question and returns what it said. The task is sent with no tool, no
// tool endpoint and no conversation, so the agent answers on its own and is given nothing of the
// vault. An agent that cannot be started, stops for a reason of its own or says nothing fails with
// port.ErrUnavailable. The end of the context stops the run and fails with the context's error.
func Answer(ctx context.Context, agent Starter, task agentport.Task) (string, error) {
	return agentport.Answer(ctx, agent, task)
}
