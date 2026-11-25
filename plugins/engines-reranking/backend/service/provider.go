package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/usecase/reranking"
)

func newProvider(r *reranking.Reranker) connection.Provider {
	provider := llm.Provide(nil, llm.Roles{
		Completer: nil, Chatter: nil, Embedder: nil, Reranker: reranker{reranking: r}, Profiler: nil, Models: r, Reasoning: nil,
	})
	provider[connection.OperationDetected] = connection.ServeDetected(r.DetectConnections)
	return provider
}
