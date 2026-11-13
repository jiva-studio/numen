package searchsourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
)

// VectorRepository holds the vectors made from chunks.
type VectorRepository interface {
	// SaveVectors writes a group of vectors, both representations of each in the
	// same write. A chunk holding one and not the other is absent from the coarse
	// pass and invisible to the question of what has no vector.
	//
	// Each vector is also stored under the hash of the text it was created from,
	// so renumbering the chunks does not require re-embedding it.
	SaveVectors(ctx context.Context, vectors []vector.Stored) error

	// GetKeptVectors returns, by hash, the vectors already created under the given
	// recipe for the texts the hashes identify. These vectors are not requested
	// from the model again.
	GetKeptVectors(ctx context.Context, recipe string, hashes [][]byte) (map[string][]byte, error)
}
