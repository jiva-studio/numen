package embeddingport

import (
	"context"
	"fmt"
	"strconv"
)

// EmbeddingModel names the model a vector came from. It is stored beside the
// vector, and a vector whose model differs from the one now configured is
// stale: its numbers describe directions another model chose.
//
// Every field but MinChunkLength and ChunkTokens is part of the identity. Two models of the same name at different
// widths produce vectors that cannot be compared, and the width is what a
// reader needs to know before it decodes stored bytes.
type EmbeddingModel struct {
	Name       string
	Dimensions int

	MaxTokens int

	Pooling string

	From string

	Handling string

	MinChunkLength int

	ChunkTokens int
}

// String is the identity as one value, for a column that holds it.
func (m EmbeddingModel) String() string {
	return m.Name + "@" + strconv.Itoa(m.Dimensions)
}

// Recipe is everything about this model that decides what a vector is, as one
// value.
//
// A vector retained beyond the row that pointed at it is looked up by the text
// it was made from and the recipe it was made under. A property missing from
// the recipe can change while the key does not, and a vector found under a key
// that does not describe it is worse than no stored vector.
//
// The provider of the vector comes first. Two places serving one name are two
// sets of rows, and a query uses the recipe the index was filled with.
//
// How the numbers are stored is the quantisation and the scale applied to it.
// The scale defines the grid of byte values, so a changed scale gives every
// stored byte a different meaning, and the recipe detects the change.
func (m EmbeddingModel) Recipe() string {
	return fmt.Sprintf("%s|%s|%d|%d|%s|%s|%s@%g",
		m.From, m.Name, m.Dimensions, m.MaxTokens, m.Pooling, m.Handling, QuantisedInt8, Int8Scale)
}

// Embedder converts text into vectors. The core requests them and does not know
// whether they are computed on this machine or obtained from a service.
//
// Vectors are unit length, so a dot product is a cosine and the sign of a
// dimension carries meaning.
type Embedder interface {
	// GetModel is the identity every vector this embedder returns belongs to.
	GetModel() EmbeddingModel
	// Embed returns one vector per text, made for the text as a document that is searched, in the order
	// the texts arrived.
	// Any number of texts may be passed: how a request to the model is
	// bounded is the implementation's business.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Concurrency is how many batches the embedder runs in parallel, and one for
	// an embedder that runs them one at a time.
	Concurrency() int

	// Close releases whatever the model holds.
	Close() error
}

// QueryEmbedder is an Embedder that makes the vectors of queries, which a model that tells a question
// from the passage that answers it reads with a prefix of its own. Embed makes those of documents.
//
// An Embedder that is no QueryEmbedder has one kind of vector, and EmbedQueries asks it for that.
type QueryEmbedder interface {
	Embedder
	// EmbedQueries returns one vector per query, in the order the queries arrived.
	EmbedQueries(ctx context.Context, queries []string) ([][]float32, error)
}

// EmbedQueries returns one vector per text, made for searching with the text as the question.
func EmbedQueries(ctx context.Context, e Embedder, queries []string) ([][]float32, error) {
	if q, ok := e.(QueryEmbedder); ok {
		return q.EmbedQueries(ctx, queries)
	}
	return e.Embed(ctx, queries)
}

// FloorAdvisor is an Embedder that reports the similarity a passage starts out
// having to reach to answer a query under its model.
//
// The hint is not part of the model's identity and is absent from Recipe.
type FloorAdvisor interface {
	// GetFloorHint returns the starting floor, and zero where the model has none.
	GetFloorHint(ctx context.Context) float64
}
