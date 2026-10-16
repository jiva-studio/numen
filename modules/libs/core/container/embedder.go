package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

// EmbedderReady reports whether an embedding plugin is active or present.
func (c Config) EmbedderReady() bool {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("embedding"); ok {
			return true
		}
	}
	return false
}

// Embedders are what makes the vectors a vault is searched by and what makes
// the vector a question is asked with.
//
// When the embedding plugin is absent or disabled, search queries seamlessly fall
// back to SQLite FTS5 full-text search without throwing errors.
func (c Config) Embedders(ctx context.Context, _ *run.Tasks) (indexing, asking port.Embedder, closer func() error, why error) {
	if host, ok := c.findPluginHost("embedding"); ok {
		bridge, err := plugin.OpenEmbedderBridge(ctx, host)
		if err != nil {
			return nil, nil, func() error { return nil }, err
		}
		return bridge, bridge, bridge.Close, nil
	}
	return nil, nil, func() error { return nil }, nil
}

// Embedder is what makes the vectors a vault is searched by, waited for.
func (c Config) Embedder(ctx context.Context) (port.Embedder, func() error, error) {
	if host, ok := c.findPluginHost("embedding"); ok {
		bridge, err := plugin.OpenEmbedderBridge(ctx, host)
		if err != nil {
			return nil, nil, err
		}
		return bridge, bridge.Close, nil
	}
	return nil, func() error { return nil }, nil
}

// OpenQuestionEmbedder is what embeds a question, for a run that fills no
// index. Only the provider that answers questions is opened.
func (c Config) OpenQuestionEmbedder(ctx context.Context) (port.Embedder, func() error, error) {
	return c.Embedder(ctx)
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
