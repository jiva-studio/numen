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
func (c Config) Embedders(ctx context.Context, _ *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error) {
	embedder, closeEmbedder, err := c.Embedder(ctx)
	if err != nil {
		return nil, nil, func() error { return nil }, err
	}
	return embedder, embedder, closeEmbedder, nil
}

// Embedder is the embedding plugin that produces the vectors of a vault. It is
// nil when no embedding plugin is enabled.
func (c Config) Embedder(ctx context.Context) (embeddingport.Embedder, func() error, error) {
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
	return bridge, bridge.Close, nil
}
