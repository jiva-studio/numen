package assembly

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/usecase/run"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

// WorkerAssembly is what this window asks for its background workers and embedding.
type WorkerAssembly interface {
	// OpenEmbedders opens the model this installation turns text into vectors
	// with, and the one a query is turned into a vector by. The two are one
	// object where the settings name one provider.
	OpenEmbedders(
		ctx context.Context, tasks *run.Tasks,
	) (indexing, asking embeddingport.Embedder, closer func() error, why error)
}
