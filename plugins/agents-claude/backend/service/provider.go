package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// Provider returns the operations the plugin serves for its connection type: llm.complete and the
// option lists llm.models and llm.reasoning.
func (a *agent) Provider() connection.Provider {
	return llm.Provide(nil, llm.Roles{Completer: a.completion, Models: a.completion, Reasoning: a.completion})
}
