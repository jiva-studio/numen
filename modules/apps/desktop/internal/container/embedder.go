package container

import (
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed/onnx"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed/openai"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Embedders are what makes the vectors a vault is searched by and what makes
// the vector a question is asked with.
//
// They are one object when the settings name one placement, so a model on this
// machine is loaded once. Naming a placement for questions is what puts a
// vault indexed over a network within reach of a machine that has none.
//
// An embedder is optional: an installation with none answers with nothing, and a
// model that cannot be opened — no file, no key, a width this build does not
// expect — is that same absence, carrying the reason with it.
//
// The closer is returned separately because a local model holds a session that
// has to be let go of, and a service holds nothing.
func (c Config) Embedders() (indexing, asking port.Embedder, close func() error, why error) {
	indexing, closeIndexing, why := c.embedder(c.Embedding.Indexing)
	if why != nil {
		return nil, nil, nil, why
	}
	if c.Embedding.Query.Use == "" {
		return indexing, indexing, closeIndexing, nil
	}

	asking, closeAsking, why := c.embedder(c.Embedding.Query)
	if why != nil {
		if closeIndexing != nil {
			_ = closeIndexing()
		}
		return nil, nil, nil, why
	}
	return indexing, asking, both(closeIndexing, closeAsking), nil
}

// Embedder is what makes the vectors a vault is searched by. A run that indexes
// and never asks needs no other.
func (c Config) Embedder() (port.Embedder, func() error, error) {
	return c.embedder(c.Embedding.Indexing)
}

func (c Config) embedder(where embed.Placement) (port.Embedder, func() error, error) {
	switch where.Use {
	case embed.UseService:
		client, err := openai.New(c.Embedding.Model, where.Service)
		if err != nil {
			return nil, nil, err
		}
		return client, func() error { return nil }, nil

	case embed.UseLocal:
		model, err := onnx.Open(c.Embedding.Model, where.Local)
		if err != nil {
			return nil, nil, err
		}
		return model, model.Close, nil
	}

	return nil, nil, nil
}

// both is one closer for two, letting go of the second whatever the first says.
func both(first, second func() error) func() error {
	return func() error {
		var why error
		if second != nil {
			why = second()
		}
		if first != nil {
			if err := first(); why == nil {
				why = err
			}
		}
		return why
	}
}
