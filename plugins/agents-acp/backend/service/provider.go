package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// Provider returns the operations the plugin serves for every connection type of it: llm.complete, the
// option lists llm.models and llm.reasoning, and acp.task.
func (s *agentServer) Provider() connection.Provider {
	provider := llm.Provide(nil, llm.Roles{Completer: s.completion, Models: s.completion, Reasoning: s.completion})
	return acp.Provide(provider, s.tasks)
}
