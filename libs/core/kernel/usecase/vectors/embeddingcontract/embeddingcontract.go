// Package embeddingcontract re-exports the embedding port contract.
package embeddingcontract

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
)

type (
	ChunkCursor      = embeddingport.ChunkCursor
	Embedder         = embeddingport.Embedder
	EmbeddingModel   = embeddingport.EmbeddingModel
	QueryEmbedder    = embeddingport.QueryEmbedder
	Vector           = embeddingport.Vector
	VectorIndex      = embeddingport.VectorIndex
	VectorQueries    = embeddingport.VectorQueries
	VectorRepository = embeddingport.VectorRepository
)

// EmbedQueries returns one vector per text, made for searching with the text as the question.
func EmbedQueries(ctx context.Context, e Embedder, queries []string) ([][]float32, error) {
	return embeddingport.EmbedQueries(ctx, e, queries)
}

const (
	Int8Scale     = embeddingport.Int8Scale
	QuantisedInt8 = embeddingport.QuantisedInt8
)
