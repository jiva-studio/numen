// Package embeddingcontract re-exports the embedding port contract.
package embeddingcontract

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
)

type (
	Embedder         = embeddingport.Embedder
	QueryEmbedder    = embeddingport.QueryEmbedder
	VectorIndex      = embeddingport.VectorIndex
	VectorQueries    = embeddingport.VectorQueries
	VectorRepository = embeddingport.VectorRepository
)
