package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// Provider returns the operations the plugin serves for every connection type of it: llm.complete, the
// option lists llm.models and llm.reasoning, acp.task, and connections.detected.
func (s *agentServer) Provider() connection.Provider {
	provider := llm.Provide(nil, llm.Roles{Completer: s.completion, Models: s.completion, Reasoning: s.completion})
	provider = acp.Provide(provider, s.tasks)
	provider[connection.OperationDetected] = connection.ServeDetected(s.detection.DetectConnections)
	return provider
}
