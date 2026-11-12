package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// Provider returns the operations the plugin serves for its connection type: llm.complete and the
// option lists llm.models and llm.reasoning.
func (s *agentServer) Provider() connection.Provider {
	return llm.Provide(nil, llm.Roles{Completer: s.completion, Models: s.completion, Reasoning: s.completion})
}
