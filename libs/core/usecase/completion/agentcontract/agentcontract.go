// Package agentcontract re-exports the agent port contract for completion use cases.
package agentcontract

import "github.com/jiva-studio/numen/libs/core/port/agentport"

type (
	Agent = agentport.Agent
	AgentModel = agentport.AgentModel
	AgentProgram = agentport.AgentProgram
	AgentProvider = agentport.AgentProvider
	CompletionPrograms = agentport.CompletionPrograms
	ReasoningLevel = agentport.ReasoningLevel
	Run = agentport.Run
	Starter = agentport.Starter
	Step = agentport.Step
	Task = agentport.Task
	ToolDeclaration = agentport.ToolDeclaration
	ToolEndpoint = agentport.ToolEndpoint
)

var (
	Answer = agentport.Answer
	GetDefaultProgram = agentport.GetDefaultProgram
)
