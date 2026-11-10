package vectors

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
)

// EmbedQueries makes the vectors of search queries with an embedder.
type EmbedQueries struct {
	documents embeddingport.Embedder
	queries   embeddingport.QueryEmbedder
}

// NewEmbedQueries makes the vectors of queries with queries where the model tells a query from a
// document, and with documents, which has one kind of vector, where queries is nil.
func NewEmbedQueries(documents embeddingport.Embedder, queries embeddingport.QueryEmbedder) *EmbedQueries {
	return &EmbedQueries{documents: documents, queries: queries}
}

// Execute returns one vector per query, made for searching with the query as the question.
func (u *EmbedQueries) Execute(ctx context.Context, queries []string) ([][]float32, error) {
	if u.queries != nil {
		return u.queries.EmbedQueries(ctx, queries)
	}
	return u.documents.Embed(ctx, queries)
}
