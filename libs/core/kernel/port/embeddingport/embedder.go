package embeddingport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
)

// Embedder converts text into vectors. The core requests them and does not know
// whether they are computed on this machine or obtained from a service.
//
// Vectors are unit length, so a dot product is a cosine and the sign of a
// dimension carries meaning.
type Embedder interface {
	// GetModel is the identity every vector this embedder returns belongs to.
	GetModel() embedding.Model
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
// An Embedder that is no QueryEmbedder has one kind of vector, and queries are embedded with Embed.
type QueryEmbedder interface {
	Embedder
	// EmbedQueries returns one vector per query, in the order the queries arrived.
	EmbedQueries(ctx context.Context, queries []string) ([][]float32, error)
}

// FloorAdvisor is an Embedder that reports the similarity a passage starts out
// having to reach to answer a query under its model.
//
// The hint is not part of the model's identity and is absent from Recipe.
type FloorAdvisor interface {
	// GetFloorHint returns the starting floor, and zero where the model has none.
	GetFloorHint(ctx context.Context) float64
}

// ModelWatcher reports when the model behind an embedder changes.
type ModelWatcher interface {
	// OnModelChange registers fn to run each time the model changes.
	OnModelChange(fn func())
}

// WatchedEmbedder is an Embedder whose model can change while the window is open, as the model of a
// plugin does when its settings change.
type WatchedEmbedder interface {
	Embedder
	ModelWatcher
}
