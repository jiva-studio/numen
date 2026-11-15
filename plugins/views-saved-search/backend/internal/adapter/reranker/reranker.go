// Package reranker scores documents against a query through the llm.rerank rows of the plugin's
// connections setting.
package reranker

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ports"
)

// Rows lists the rows of the plugin's connections setting and calls through one of them.
// *connection.Client satisfies it.
type Rows interface {
	ListRows(ctx context.Context, operation string) ([]connection.Row, error)
	Prefer(row connection.PreferredRow) connection.Caller
}

// Reranker asks the rows of the plugin's connections setting for llm.rerank, one after another.
type Reranker struct {
	rows Rows
}

var _ ports.Reranker = (*Reranker)(nil)

// New returns a Reranker over the rows.
func New(rows Rows) *Reranker {
	return &Reranker{rows: rows}
}

// Rerank asks each row in order, alone, and answers the scores of the first row that answers with that
// row. It returns ports.ErrNoReranker when no row serves llm.rerank, the error of the context when it ends,
// and the failures of every row, by connection name, when none answers.
func (r *Reranker) Rerank(ctx context.Context, request ports.RerankRequest) (ports.Reranked, error) {
	rows, err := r.rows.ListRows(ctx, llm.OperationRerank)
	if err != nil {
		return ports.Reranked{}, err
	}
	if len(rows) == 0 {
		return ports.Reranked{}, ports.ErrNoReranker
	}
	message := &llm.RerankRequest{Query: request.Query, Instruction: request.Instruction, Documents: request.Documents}
	var failures []error
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return ports.Reranked{}, err
		}
		caller := r.rows.Prefer(connection.PreferredRow{ConnectionID: row.ConnectionID, Options: row.Options, IsExclusive: true})
		result, err := llm.Rerank(ctx, caller, message, request.Priority)
		if err == nil {
			return ports.Reranked{Scores: convertScores(result.GetScores()), Reranker: newRerankerID(row)}, nil
		}
		if ctx.Err() != nil {
			return ports.Reranked{}, ctx.Err()
		}
		failures = append(failures, fmt.Errorf("%s: %w", row.ConnectionName, err))
	}
	return ports.Reranked{}, errors.Join(failures...)
}

// ListRerankers returns the rows of llm.rerank in the order Rerank tries them.
func (r *Reranker) ListRerankers(ctx context.Context) ([]ports.RerankerID, error) {
	rows, err := r.rows.ListRows(ctx, llm.OperationRerank)
	if err != nil {
		return nil, err
	}
	listed := make([]ports.RerankerID, len(rows))
	for i, row := range rows {
		listed[i] = newRerankerID(row)
	}
	return listed, nil
}

func newRerankerID(row connection.Row) ports.RerankerID {
	return ports.RerankerID{ConnectionID: row.ConnectionID, Options: row.Options}
}

func convertScores(scores []float32) []float64 {
	converted := make([]float64, len(scores))
	for i, score := range scores {
		converted[i] = float64(score)
	}
	return converted
}
