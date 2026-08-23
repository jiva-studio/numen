package container

import (
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/window"
)

// Cutting is how a source's text is cut into chunks.
//
// It is a fact about the settings and not about what is loaded: the sizes are
// part of what a chunk is kept under, and a chunk cut one way in a window and
// another way in a terminal is a source cut again every time the two take
// turns. This is where the number comes from, and there is nowhere else to
// take it from.
func (c Config) Cutting() window.Sizes {
	if c.Embedding.Indexing.Use == "" {
		return window.Sizes{}
	}
	return window.Sizes{Limit: window.Under(c.Embedding.Model.MaxTokens)}
}

// Extract cuts a vault's sources into chunks. Every entry point takes it from
// here, so what a chunk is kept under is one answer.
func (c Config) Extract(sources port.SourceRepository, owing port.SourceQueries, v domain.Vault) (source.Extract, error) {
	derived, err := c.DerivedStores().Open(v)
	if err != nil {
		return source.Extract{}, err
	}
	return source.Extract{
		Readers:      c.VaultReaders(),
		Sources:      sources,
		Owing:        owing,
		Derived:      derived,
		Sizes:        c.Cutting(),
		RebuildIndex: c.RebuildIndex,
	}, nil
}
