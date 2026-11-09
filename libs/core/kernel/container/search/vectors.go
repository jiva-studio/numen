package search

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/completionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

// EmbedderReady reports whether an embedding plugin is enabled.
func (c Config) EmbedderReady() bool {
	if c.ResolvePlugin == nil {
		return false
	}
	_, err := c.ResolvePlugin(bridge.CapabilityEmbedding)
	return err == nil
}

// Embedders returns the embedder for the vectors a vault is searched by and the
// embedder for the vector of a question; the embedding plugin provides both.
// An installation with no embedding plugin enabled has neither, and no error:
// search answers on the full-text index alone.
func (c Config) Embedders(ctx context.Context, tasks *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error) {
	embedder, closeEmbedder, err := c.OpenEmbedder(ctx, NewModelReadReporter(tasks))
	if err != nil {
		return nil, nil, func() error { return nil }, err
	}
	return embedder, embedder, closeEmbedder, nil
}

// Embedder is the embedding plugin that produces the vectors of a vault. It is
// nil when no embedding plugin is enabled.
func (c Config) Embedder(ctx context.Context) (embeddingport.Embedder, func() error, error) {
	return c.OpenEmbedder(ctx, nil)
}

// readingTheModel is the ID of the task that reports a read of the embedding model that failed.
const readingTheModel = "read embedding model"

// NewModelReadReporter returns a callback that records an embedding model read failure in tasks.
func NewModelReadReporter(tasks *run.Tasks) func(error) {
	if tasks == nil {
		return nil
	}
	return func(err error) {
		tasks.Set(run.Task{ID: readingTheModel, Title: "Read embedding model", Error: err.Error()})
	}
}

// OpenEmbedder opens the active embedding plugin when available.
func (c Config) OpenEmbedder(ctx context.Context, onReadError func(error)) (embeddingport.Embedder, func() error, error) {
	if c.ResolvePlugin == nil {
		return nil, func() error { return nil }, nil
	}
	host, err := c.ResolvePlugin(bridge.CapabilityEmbedding)
	if errors.Is(err, completionport.ErrUnavailable) {
		return nil, func() error { return nil }, nil
	}
	if err != nil {
		return nil, nil, err
	}
	opened, err := embedderbridge.Open(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	if onReadError != nil {
		opened.OnModelReadError(onReadError)
	}
	FollowChunkTokens(c.Chunks, opened, opened.OnModelChange)
	return opened, opened.Close, nil
}

// FollowChunkTokens updates cell whenever embedder reports a new chunk token limit.
func FollowChunkTokens(cell *ChunkTokens, embedder embeddingport.Embedder, onChange func(func())) {
	cell.Set(embedder.GetModel().ChunkTokens)
	onChange(func() { cell.Set(embedder.GetModel().ChunkTokens) })
}
