package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// Provider returns the operations the plugin serves for its connection type: llm.complete, the option
// lists llm.models and llm.reasoning, and acp.task.
func (a *agent) Provider() connection.Provider {
	provider := llm.Provide(nil, llm.Roles{Completer: a.completion, Models: a.completion, Reasoning: a.completion})
	return acp.Provide(provider, a.tasks)
}
