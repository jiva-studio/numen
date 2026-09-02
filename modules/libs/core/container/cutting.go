package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/cutting"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Cutting is how a source's text is cut into chunks.
//
// It is a fact about the settings and not about what is loaded: the sizes are
// part of what a chunk is kept under, and a chunk cut one way in a window and
// another way in a terminal is a source cut again every time the two take
// turns. This is where the number comes from, and there is nowhere else to
// take it from.
func (c Config) Cutting() cutting.Sizes {
	if c.Embedding.Indexing.Use == "" {
		return cutting.Sizes{}
	}
	return cutting.Sizes{Limit: cutting.Under(c.Embedding.Model.MaxTokens)}
}

// NotesCutAt is the note repository, told the sizes a note is cut at. A note is
// cut in the adapter that stores it, and the sizes reach that adapter from here.
func (i *Index) NotesCutAt(sizes cutting.Sizes) port.NoteRepository {
	return i.db.Notes().Cut(sizes)
}

// Scan is the walk that reads a vault's notes into the index, cut at this
// installation's sizes. Every application that reads a vault takes it from here.
func (c Config) Scan(db *Index) vault.Scan {
	return vault.Scan{
		Readers:      c.VaultReaders(),
		Vaults:       db.Vaults(),
		Notes:        db.NotesCutAt(c.Cutting()),
		Known:        db.Queries(),
		Maintenance:  db.Maintenance(),
		RebuildIndex: c.RebuildIndex,
	}
}

// Searchable is what makes a vault answer, put together the one way: the notes
// read, the books read, and the vectors made. Every entry point takes it from
// here, so a vault made searchable in a terminal and a vault made searchable in
// a window are the same vault.
func (c Config) Searchable(ctx context.Context, db *Index, embedder port.Embedder, v domain.Vault) (vault.Searchable, error) {
	// The coarse index is built for one width, and the width is the model's. A
	// vault made searchable is a vault whose vector index holds what the model
	// makes, whichever entry point is doing the making.
	if embedder != nil {
		model := embedder.Model()
		if err := db.FitVectors(ctx, model.Dimensions, model.Recipe()); err != nil {
			return vault.Searchable{}, err
		}
	}
	books, err := c.Extract(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		return vault.Searchable{}, err
	}
	vectors, err := c.Embed(db, embedder, v)
	if err != nil {
		return vault.Searchable{}, err
	}
	return vault.Searchable{
		Notes:   c.Scan(db),
		Books:   books,
		Vectors: vectors,
	}, nil
}

// Embed gives a vault's chunks the vectors they owe.
func (c Config) Embed(db *Index, embedder port.Embedder, v domain.Vault) (source.Embed, error) {
	derived, err := c.DerivedStores().Open(v)
	if err != nil {
		return source.Embed{}, err
	}
	return source.Embed{
		Readers:   c.VaultReaders(),
		Derived:   derived,
		Documents: c.Documents(),
		Chunks:    db.VectorsOwing(),
		Vectors:   db.Vectors(),
		Embedder:  embedder,
	}, nil
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
		Documents:    c.Documents(),
		Sizes:        c.Cutting(),
		RebuildIndex: c.RebuildIndex,
	}, nil
}
