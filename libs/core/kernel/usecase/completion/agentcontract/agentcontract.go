// Package agentcontract re-exports the agent port contract for completion use cases.
package agentcontract

import (
	"context"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

type (
	Agent              = agentport.Agent
	AgentProvider      = agentport.AgentProvider
	CompletionPrograms = agentport.CompletionPrograms
	Run                = agentport.Run
	Starter            = agentport.Starter
)

func Answer(ctx context.Context, agent agentport.Starter, task agentdomain.Task) (string, error) {
	return agentport.Answer(ctx, agent, task)
}

func GetDefaultProgram(programs []agentdomain.Program) (agentdomain.Program, bool) {
	return agentport.GetDefaultProgram(programs)
}
