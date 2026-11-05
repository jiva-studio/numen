package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/libs/core/usecase/run"

	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

// EmbedderReady reports whether an embedding plugin is enabled.
func (c Config) EmbedderReady() bool {
	return c.hasPlugin(bridge.CapabilityEmbedding)
}

// Embedders returns the embedder that produces the vectors a vault is searched
// by and the embedder that produces the vector for a question: the embedding
// plugin provides both.
//
// An installation with no embedding plugin enabled has neither, and no error:
// search answers on the words of the full-text index alone.
func (c Config) Embedders(ctx context.Context, tasks *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error) {
	embedder, closeEmbedder, err := c.openEmbedder(ctx, newModelReadReporter(tasks))
	if err != nil {
		return nil, nil, func() error { return nil }, err
	}
	return embedder, embedder, closeEmbedder, nil
}

// Embedder is the embedding plugin that produces the vectors of a vault. It is
// nil when no embedding plugin is enabled.
func (c Config) Embedder(ctx context.Context) (embeddingport.Embedder, func() error, error) {
	return c.openEmbedder(ctx, nil)
}

// readingTheModel is the ID of the task that reports a read of the embedding model that failed.
const readingTheModel = "read embedding model"

// newModelReadReporter reports a read of the embedding model that fails after a setting of the plugin
// as a task with the failure as its reason. It reports nothing where there are no tasks.
func newModelReadReporter(tasks *run.Tasks) func(error) {
	if tasks == nil {
		return nil
	}
	return func(err error) {
		tasks.Set(run.Task{ID: readingTheModel, Title: "Read embedding model", Error: err.Error()})
	}
}

// openEmbedder opens the embedding plugin. onReadError, when set, receives each failed read of its model.
func (c Config) openEmbedder(ctx context.Context, onReadError func(error)) (embeddingport.Embedder, func() error, error) {
	host, err := c.resolvePlugin(bridge.CapabilityEmbedding)
	if errors.Is(err, completionport.ErrUnavailable) {
		return nil, func() error { return nil }, nil
	}
	if err != nil {
		return nil, nil, err
	}
	bridge, err := embedderbridge.Open(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	if onReadError != nil {
		bridge.OnModelReadError(onReadError)
	}
	followChunkTokens(c.Chunks, bridge, bridge.OnModelChange)
	return bridge, bridge.Close, nil
}

// followChunkTokens holds the chunk size the embedder reports in the cell, and again each time the
// embedder announces a change of its model.
func followChunkTokens(cell *ChunkTokens, embedder embeddingport.Embedder, onChange func(func())) {
	cell.Set(embedder.GetModel().ChunkTokens)
	onChange(func() { cell.Set(embedder.GetModel().ChunkTokens) })
}
