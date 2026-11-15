// Package reranker scores documents against a query through the llm.rerank connections of the plugin.
package reranker

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ports"
)

// Reranker asks the rows of the plugin's connections setting, in order, for llm.rerank.
type Reranker struct {
	caller connection.Caller
}

// New returns a Reranker that calls through caller.
func New(caller connection.Caller) *Reranker {
	return &Reranker{caller: caller}
}

// Rerank returns one score for each document, in the order of the documents. It returns
// ports.ErrNoReranker when no row serves llm.rerank, and every other failure as the connection reports it.
func (r *Reranker) Rerank(ctx context.Context, query, instruction string, documents []string) ([]float64, error) {
	result, err := llm.Rerank(ctx, r.caller, &llm.RerankRequest{Query: query, Instruction: instruction, Documents: documents}, connection.PriorityInteractive)
	if errors.Is(err, connection.ErrNoConnection) {
		return nil, ports.ErrNoReranker
	}
	if err != nil {
		return nil, err
	}
	scores := make([]float64, len(result.GetScores()))
	for i, score := range result.GetScores() {
		scores[i] = float64(score)
	}
	return scores, nil
}
