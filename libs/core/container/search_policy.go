package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostsearch"
	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
)

// NewFloorPolicy is the floor a search by meaning applies under the model of an embedder: the floor
// stored for the vault, else the model's hint, else search.DefaultFloor.
func (c Config) NewFloorPolicy(db *Index, embedder embeddingport.Embedder, errorHandler func(error)) search.FloorPolicy {
	return search.NewFloorPolicy(db.SearchFloors(), getFloorHint(embedder), errorHandler)
}

// SetPluginSearch lends the plugins the search index, the model that fills it and the floors of the
// vaults. Without an embedder the plugins are informed that the installation has no model.
func (c Config) SetPluginSearch(db *Index, embedder embeddingport.Embedder, errorHandler func(error)) {
	if c.PluginHost == nil {
		return
	}
	var host pluginport.PluginSearchHost = c.PluginHost
	host.SetPluginSearch(hostsearch.Search{
		Embedder: embedder,
		Samples:  db.SearchSamples(),
		Nearest:  db.Nearest(),
		Progress: db.Progress(),
		Policy:   c.NewFloorPolicy(db, embedder, errorHandler),
	})
}

// getFloorHint is the starting floor an embedder reports for its model, and nil for one that reports
// none.
func getFloorHint(embedder embeddingport.Embedder) search.FloorHint {
	advisor, ok := embedder.(embeddingport.FloorAdvisor)
	if !ok {
		return nil
	}
	return func(ctx context.Context) float64 { return advisor.GetFloorHint(ctx) }
}
