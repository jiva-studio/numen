package port

import "context"

// QuantisedInt8 names the quantisation of a stored vector's bytes: one signed
// byte per dimension, at the scale the model is written down with. It is
// recorded because a blob does not say what it holds.
const QuantisedInt8 = "int8"

// Vector is one chunk's embedding, in both representations that are stored.
//
// Value is the rerank's copy, quantised as Kind says. Coarse is one bit per
// dimension, which is the coarse pass's whole question. Fingerprint addresses
// the text the model read, which is what the vector is kept under once the
// chunk that pointed at it is gone.
type Vector struct {
	ChunkID     int64
	Fingerprint []byte
	Model       EmbeddingModel
	Kind        string
	Value       []byte
	Coarse      []byte
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

	// Kept is the vectors already made under the recipe given for the texts the
	// fingerprints address, by fingerprint. What comes back was paid for once
	// and is not asked of a model again.
	Kept(ctx context.Context, recipe string, fingerprints [][]byte) (map[string][]byte, error)
}
