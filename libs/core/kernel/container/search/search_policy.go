package search

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hostsearch"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
)

// NewFloorPolicy is the floor a search by meaning applies under the model of an embedder: the floor
// stored for the vault, else the model's hint, else search.DefaultFloor.
func (c Config) NewFloorPolicy(db *vault.Index, embedder embeddingport.Embedder, errorHandler func(error)) searchusecase.FloorPolicy {
	return searchusecase.NewFloorPolicy(db.SearchFloors(), GetFloorHint(embedder), errorHandler)
}

// SetPluginSearch lends the plugins the search index, the model that fills it and the floors of the
// vaults. Without an embedder the plugins are informed that the installation has no model.
func (c Config) SetPluginSearch(db *vault.Index, embedder embeddingport.Embedder, errorHandler func(error)) {
	if c.PluginHost == nil {
		return
	}
	var host pluginport.PluginSearchHost = c.PluginHost
	host.SetPluginSearch(hostsearch.Search{
		Embedder: embedder,
		Queries:  NewEmbedQueries(embedder),
		Samples:  db.SearchSamples(),
		Nearest:  db.Nearest(),
		Progress: db.Progress(),
		Stats:    hostsearch.NewStats(db.Stats(), c.Clock),
		Policy:   c.NewFloorPolicy(db, embedder, errorHandler),
		Passages: searchusecase.NewSearchPassages(c.NewSearch(db, embedder, errorHandler)),
	})
}

// NewEmbedQueries makes the vectors of queries with the embedder, as queries where its model tells a
// query from a document.
func NewEmbedQueries(embedder embeddingport.Embedder) *vectorscase.EmbedQueries {
	queries, ok := embedder.(embeddingport.QueryEmbedder)
	if !ok {
		return vectorscase.NewEmbedQueries(embedder, nil)
	}
	return vectorscase.NewEmbedQueries(embedder, queries)
}

// GetFloorHint returns the similarity floor hint reported by embedder.
func GetFloorHint(embedder embeddingport.Embedder) searchusecase.FloorHint {
	advisor, ok := embedder.(embeddingport.FloorAdvisor)
	if !ok {
		return nil
	}
	return func(ctx context.Context) float64 { return advisor.GetFloorHint(ctx) }
}
