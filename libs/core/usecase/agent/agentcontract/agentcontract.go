// Package agentcontract re-exports the agent port contract for agent use cases.
package agentcontract

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

type (
	Agent           = agentport.Agent
	AgentModel      = agentport.AgentModel
	AgentProgram    = agentport.AgentProgram
	AgentProvider   = agentport.AgentProvider
	ReasoningLevel  = agentport.ReasoningLevel
	Run             = agentport.Run
	Starter         = agentport.Starter
	Step            = agentport.Step
	Task            = agentport.Task
	ToolDeclaration = agentport.ToolDeclaration
	ToolEndpoint    = agentport.ToolEndpoint
)

func Answer(ctx context.Context, agent agentport.Starter, task agentport.Task) (string, error) {
	return agentport.Answer(ctx, agent, task)
}

func GetDefaultProgram(programs []agentport.AgentProgram) (agentport.AgentProgram, bool) {
	return agentport.GetDefaultProgram(programs)
}
