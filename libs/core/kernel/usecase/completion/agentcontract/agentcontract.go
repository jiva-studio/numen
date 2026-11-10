// Package agentcontract re-exports the agent port contract for completion use cases.
package agentcontract

import (
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

type (
	Agent              = agentport.Agent
	AgentProvider      = agentport.AgentProvider
	CompletionPrograms = agentport.CompletionPrograms
	Run                = agentport.Run
)

func GetDefaultProgram(programs []agentdomain.Program) (agentdomain.Program, bool) {
	return agentport.GetDefaultProgram(programs)
}
