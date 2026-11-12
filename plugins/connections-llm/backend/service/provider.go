package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/completion"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/embedding"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/models"
)

// provider answers the calls of the provider service with the use cases of the plugin.
type provider struct {
	completion *completion.Completion
	embedding  *embedding.Embedding
	models     *models.Models
}

// roles returns the operations of the llm contract the plugin serves.
func (p *provider) roles() connection.Provider {
	return llm.Provide(nil, llm.Roles{
		Completer: p.completion, Chatter: p.completion, Embedder: p.embedding, Profiler: p.embedding,
		Models: p.models, Reasoning: p.models,
	})
}
