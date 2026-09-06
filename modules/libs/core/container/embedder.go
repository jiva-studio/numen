package container

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed/onnx"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed/openai"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/embedders"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

// Embedders are what makes the vectors a vault is searched by and what makes
// the vector a question is asked with.
//
// They are one model where the settings name one provider. Naming a provider
// for questions is what puts a vault indexed over a network within reach of a
// machine that has none, and the two are held to being one model behind this.
//
// An embedder is optional: an installation naming no provider answers with
// nothing. A provider that cannot be built — no key, no base URL — is a
// reason, and nothing is built at all.
func (c Config) Embedders(ctx context.Context, tasks *task.Tasks) (indexing, asking port.Embedder, close func() error, why error) {
	first, why := c.provider(c.Embedding.Indexing)
	if why != nil {
		return nil, nil, nil, why
	}
	second, why := c.provider(c.Embedding.Query)
	if why != nil {
		return nil, nil, nil, why
	}
	indexing, asking, close = embedders.Open(ctx, tasks, first, second)
	return indexing, asking, close, nil
}

// Embedder is what makes the vectors a vault is searched by, waited for. A run
// with nowhere to show that a model is arriving waits for it instead.
func (c Config) Embedder(ctx context.Context) (port.Embedder, func() error, error) {
	held, err := c.provider(c.Embedding.Indexing)
	if err != nil {
		return nil, nil, err
	}
	embedder, close := embedders.One(ctx, held)
	return embedder, close, nil
}

// Asking is what embeds a question, for a run that fills no index. Only the
// provider that answers questions is opened, and it answers under the identity
// the index is filled with.
func (c Config) Asking(ctx context.Context) (port.Embedder, func() error, error) {
	held, err := c.provider(c.Embedding.Asking())
	if err != nil {
		return nil, nil, err
	}
	embedder, close := embedders.One(ctx, held)
	return embedder, close, nil
}

// Searching is the search a question is answered by, put together the one way:
// the passages, the vault's files, the text read out of books, and the model a
// question is embedded by.
//
// trouble is where a half that could not run is said. A search short of the
// half that asks by meaning is a search the words answer.
func (c Config) Searching(db *Index, asking port.Embedder, trouble func(error)) search.Search {
	return c.SearchingOver(db.Passages(), asking, trouble)
}

// SearchingOver is that search over the passages given, for a run that holds
// the index open for asking alone.
func (c Config) SearchingOver(passages port.PassageQueries, asking port.Embedder, trouble func(error)) search.Search {
	return search.New(passages, c.VaultReaders(), c.DerivedStores(), c.TextExtractor(),
		asking, c.Embedding.Floor, trouble)
}

// provider is which adapter answers for one half of the work, under the
// identity the index is filled with and the name that half is reached by.
//
// Nothing for a provider that names neither kind. A word that is neither is a
// word nobody implements: left to mean nothing, it is a vault searched by its
// words and no reason given.
func (c Config) provider(where embed.Provider) (embedders.Provider, error) {
	is := c.Embedding.Stored()
	switch where.Use {
	case embed.UseService:
		service, _ := where.Service()
		client, err := openai.New(is, service)
		if err != nil {
			return embedders.Provider{}, err
		}
		return embedders.Reached(is, service.Name, client), nil

	case embed.UseLocal:
		local, _ := where.Local()
		open := func(ctx context.Context, tell func(done, total int64)) (port.Embedder, error) {
			model, err := onnx.Open(ctx, is, local, tell)
			if err != nil {
				return nil, err
			}
			return model, nil
		}
		return embedders.Fetched(is, local.Name, open), nil

	case "":
		return embedders.Provider{}, nil
	}
	return embedders.Provider{}, fmt.Errorf("vectors are made %q, and they are made %q or %q",
		where.Use, embed.UseLocal, embed.UseService)
}
