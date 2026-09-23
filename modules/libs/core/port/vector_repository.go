package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// QuantisedInt8 names the quantisation of a stored vector's bytes: one signed
// byte per dimension, at the scale the model is written down with. It is
// recorded because a blob does not say what it holds.
const QuantisedInt8 = "int8"

// Int8Scale is what a stored byte of 127 stands for. Bytes on two scales are
// numbers on two grids, and the recipe carries the scale they were made at.
//
// One number serves every width. A wider vector spends fewer of the 127 levels,
// and what that costs a cosine does not grow with the width. What 0.4 buys is
// headroom: a component of a unit-length vector beyond it is clamped.
const Int8Scale = 0.4

// Vector is one chunk's embedding, in both representations that are stored.
//
// Value is the rerank's copy, quantised as Kind says. Coarse is one bit per
// dimension, which is the coarse pass's whole question. Hash is the digest of
// the text the model read, which is what the vector is kept under once the
// chunk that pointed at it is gone.
type Vector struct {
	ChunkID domain.ChunkID
	Hash    []byte
	Model   EmbeddingModel
	Kind    string
	Value   []byte
	Coarse  []byte
}

// VectorRepository holds the vectors made from chunks.
type VectorRepository interface {
	// SaveVectors writes a group of vectors, both representations of each in the
	// same write. A chunk holding one and not the other is absent from the coarse
	// pass and invisible to the question of what has no vector.
	//
	// Each is also written where a vector is kept by the address of the text it
	// was made from, so that renumbering the chunks does not buy it again.
	SaveVectors(ctx context.Context, vectors []Vector) error

	// GetKeptVectors is the vectors already made under the recipe given for the
	// texts the hashes address, by hash. What comes back was paid for once and
	// is not asked of a model again.
	GetKeptVectors(ctx context.Context, recipe string, hashes [][]byte) (map[string][]byte, error)
}
