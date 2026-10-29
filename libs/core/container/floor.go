package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/usecase/floor"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
)

// NewFloors is the floor use cases of one embedder over the cache: what the
// floor is, measuring it, setting it, and probing a query against it.
func (c Config) NewFloors(db *Index, embedder embeddingport.Embedder, errorHandler func(error)) floor.Scenarios {
	return floor.NewScenarios(floor.Config{
		Store:        db.Floors(),
		Samples:      db.FloorSamples(),
		Nearest:      db.Nearest(),
		Embedder:     embedder,
		Clock:        c.Clock(),
		FileFloor:    c.Embedding.Floor,
		DefaultFloor: search.DefaultFloor,
		Hint:         getFloorHint(embedder),
		ErrorHandler: errorHandler,
	})
}

// getFloorHint is the starting floor an embedder reports for its model, and nil
// for one that reports none.
func getFloorHint(embedder embeddingport.Embedder) floor.Hint {
	advisor, ok := embedder.(embeddingport.FloorAdvisor)
	if !ok {
		return nil
	}
	return func(ctx context.Context) float64 { return advisor.GetFloorHint(ctx) }
}
