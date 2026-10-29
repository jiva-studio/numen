package embeddingport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// QuantisedInt8 names the quantisation of a stored vector's bytes: one signed
// byte per dimension, at the scale the model is written down with. It is
// recorded because a blob does not say what it holds.
const QuantisedInt8 = "int8"

// Int8Scale is the value that a stored byte of 127 represents. Bytes on two
// scales lie on two different grids, so the recipe records the scale they were
// created at.
//
// One value serves every vector width. A wider vector uses fewer of the 127
// levels, and the resulting cosine error does not grow with the width. The
// value 0.4 sets the headroom: a component of a unit-length vector beyond it is
// clamped.
const Int8Scale = 0.4

// Vector is one chunk's embedding, in both representations that are stored.
//
// Value is the copy used for reranking, quantised as Kind specifies. Coarse is
// one bit per dimension, used by the coarse pass. Hash is the digest of the
// text the model read; the vector is stored under it after the chunk that
// referenced it is gone.
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
	// Each vector is also stored under the hash of the text it was created from,
	// so renumbering the chunks does not require re-embedding it.
	SaveVectors(ctx context.Context, vectors []Vector) error

	// GetKeptVectors returns, by hash, the vectors already created under the given
	// recipe for the texts the hashes identify. These vectors are not requested
	// from the model again.
	GetKeptVectors(ctx context.Context, recipe string, hashes [][]byte) (map[string][]byte, error)
}
