package port

import (
	"context"
	"strconv"
)

// EmbeddingModel names the model a vector came from. It is stored beside the
// vector, and a vector whose model differs from the one now configured is
// stale: its numbers describe directions another model chose.
//
// Both fields are part of the identity. Two models of the same name at
// different widths produce vectors that cannot be compared, and the width is
// what a reader needs to know before it decodes stored bytes.
type EmbeddingModel struct {
	Name       string
	Dimensions int
}

// String is the identity as one value, for a column that holds it.
func (m EmbeddingModel) String() string {
	return m.Name + "@" + strconv.Itoa(m.Dimensions)
}

// Embedder turns text into vectors. The core asks for it and does not know
// whether the answer is computed on this machine or bought from a service.
//
// Vectors are unit length, so a dot product is a cosine and the sign of a
// dimension carries meaning.
type Embedder interface {
	// Model is the identity every vector this embedder returns belongs to.
	Model() EmbeddingModel
	// Embed returns one vector per text, in the order the texts arrived.
	// Any number of texts may be passed: how a request to the model is
	// bounded is the implementation's business.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
