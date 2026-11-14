package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/completion"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/embedding"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/models"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/reranking"
)

// provider answers the calls of the provider service with the use cases of the plugin.
type provider struct {
	completion *completion.Completion
	embedding  *embedding.Embedding
	reranking  *reranking.Reranking
	models     *models.Models
}

func (p *provider) roles() connection.Provider {
	return llm.Provide(nil, llm.Roles{
		Completer: p.completion, Chatter: p.completion, Embedder: p.embedding, Reranker: p.reranking, Profiler: p.embedding,
		Models: p.models, Reasoning: p.models,
	})
}
