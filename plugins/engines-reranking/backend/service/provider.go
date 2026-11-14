package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/usecase/reranking"
)

// newProvider returns the operations of the local-reranker type: llm.rerank, llm.models and
// connections.detected.
func newProvider(r *reranking.Reranking) connection.Provider {
	provider := llm.Provide(nil, llm.Roles{
		Completer: nil, Chatter: nil, Embedder: nil, Reranker: r, Profiler: nil, Models: r, Reasoning: nil,
	})
	provider[connection.OperationDetected] = connection.ServeDetected(r.DetectConnections)
	return provider
}
