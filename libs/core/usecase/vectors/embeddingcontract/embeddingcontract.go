// Package embeddingcontract re-exports the embedding port contract.
package embeddingcontract

import "github.com/jiva-studio/numen/libs/core/port/embeddingport"

type (
	ChunkCursor      = embeddingport.ChunkCursor
	Embedder         = embeddingport.Embedder
	EmbeddingModel   = embeddingport.EmbeddingModel
	Vector           = embeddingport.Vector
	VectorIndex      = embeddingport.VectorIndex
	VectorQueries    = embeddingport.VectorQueries
	VectorRepository = embeddingport.VectorRepository
)

const (
	Int8Scale     = embeddingport.Int8Scale
	QuantisedInt8 = embeddingport.QuantisedInt8
)
