package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

// EmbedderReady reports whether an embedding plugin is enabled.
func (c Config) EmbedderReady() bool {
	return c.hasPlugin(plugin.CapabilityEmbedding)
}

// Embedders are what makes the vectors a vault is searched by and what makes
// the vector a question is asked with.
//
// An installation with no embedding plugin enabled has neither, and no error:
// search answers on the words of the full-text index alone.
func (c Config) Embedders(ctx context.Context, _ *run.Tasks) (indexing, asking port.Embedder, closer func() error, why error) {
	embedder, closeEmbedder, err := c.Embedder(ctx)
	if err != nil {
		return nil, nil, func() error { return nil }, err
	}
	return embedder, embedder, closeEmbedder, nil
}

// Embedder is what makes the vectors a vault is searched by, waited for. It is
// nil where no embedding plugin is enabled.
func (c Config) Embedder(ctx context.Context) (port.Embedder, func() error, error) {
	host, err := c.resolvePlugin(plugin.CapabilityEmbedding)
	if errors.Is(err, port.ErrUnavailable) {
		return nil, func() error { return nil }, nil
	}
	if err != nil {
		return nil, nil, err
	}
	bridge, err := plugin.OpenEmbedderBridge(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	return bridge, bridge.Close, nil
}

// NewSearch is the search a question is answered by, put together the one way:
// the passages, the vault's files, the text read out of books, and the model a
// question is embedded by.
func (c Config) NewSearch(db *Index, asking port.Embedder, errorHandler func(error)) search.Search {
	return c.NewSearchOver(db.Passages(), asking, errorHandler)
}

// NewSearchOver is that search over the passages given, for a run that holds
// the index open for asking alone.
func (c Config) NewSearchOver(passages port.PassageQueries, asking port.Embedder, errorHandler func(error)) search.Search {
	return search.New(passages, c.VaultReaders(), c.GetDerivedStores(), c.TextExtractor(),
		asking, c.Embedding.Floor, errorHandler).WithExtractors(c.ResolvedCompute.Workers)
}
