package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Chunking is how a source's text is cut into chunks.
//
// It is a fact about the settings and not about what is loaded: the sizes are
// part of what a chunk is kept under, and a chunk cut one way in a window and
// another way in a terminal is a source cut again every time the two take
// turns. This is where the number comes from, and there is nowhere else to
// take it from.
func (c Config) Chunking() chunking.Sizes {
	if c.Embedding.Indexing.Use == "" {
		return chunking.Sizes{}
	}
	return chunking.Sizes{Limit: chunking.Under(c.Embedding.Model.MaxTokens)}
}

// Legibility is what a chunk has to read like to be indexed. It comes from here
// for the reason the sizes do: one answer for one installation.
func (c Config) Legibility() chunking.Legibility {
	return chunking.Legibility{}
}

// NotesCutAt is the note repository, told the sizes a note is cut at and what
// makes one legible. A note is cut in the adapter that stores it, and both
// reach that adapter from here.
func (i *Index) NotesCutAt(sizes chunking.Sizes, reads chunking.Legibility) port.NoteRepository {
	return i.db.Notes().Cut(sizes, reads)
}

// Scan is the walk that reads a vault's notes into the index, cut at this
// installation's sizes. Every application that reads a vault takes it from here.
func (c Config) Scan(db *Index) vault.Scan {
	scan := vault.NewScan(
		c.VaultReaders(),
		db.Vaults(),
		db.NotesCutAt(c.Chunking(), c.Legibility()),
		db.Queries(),
		db.Maintenance(),
	)
	scan.Walks = db.Walks()
	scan.RebuildIndex = c.RebuildIndex
	return scan
}

// ReadWholeVault is what makes a vault answer, put together the one way: the
// notes read, the books read, and the vectors made. Every entry point takes it
// from here, so a vault made searchable in a terminal and a vault made
// searchable in a window are the same vault.
func (c Config) ReadWholeVault(
	ctx context.Context, db *Index, embedder port.Embedder, v domain.Vault,
) (vault.ReadWholeVault, error) {
	// The coarse index is built for one width, and the width is the model's. A
	// vault made searchable is a vault whose vector index holds what the model
	// makes, whichever entry point is doing the making.
	if embedder != nil {
		model := embedder.Model()
		if err := db.FitVectors(ctx, model.Dimensions, model.Recipe()); err != nil {
			return vault.ReadWholeVault{}, err
		}
		// Building the index again is where the vectors of a recipe nobody asks
		// for any more go. Every other run keeps them, so a model set back is a
		// model whose vectors are all still here.
		if c.RebuildIndex {
			if _, err := db.ForgetOtherRecipes(ctx, model.Recipe()); err != nil {
				return vault.ReadWholeVault{}, err
			}
			// The rows are gone whether or not the file gives its pages back,
			// so a vacuum that could not run is said and not waited for.
			if err := db.Compact(ctx); err != nil {
				c.trouble(err)
			}
		}
	}
	books, err := c.Extract(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		return vault.ReadWholeVault{}, err
	}
	vectors, err := c.Embed(db, embedder, v)
	if err != nil {
		return vault.ReadWholeVault{}, err
	}
	return vault.NewReadWholeVault(c.Scan(db), books, vectors), nil
}

// Embed gives a vault's chunks the vectors they owe.
func (c Config) Embed(db *Index, embedder port.Embedder, v domain.Vault) (source.Embed, error) {
	derived, err := c.DerivedStores().Open(v)
	if err != nil {
		return source.Embed{}, err
	}
	embed := source.NewEmbed(c.VaultReaders(), db.VectorsOwing(), db.Vectors())
	embed.Derived = derived
	embed.Documents = c.TextExtractor()
	embed.Embedder = embedder
	return embed, nil
}

// Extract cuts a vault's sources into chunks. Every entry point takes it from
// here, so what a chunk is kept under is one answer.
func (c Config) Extract(sources port.SourceRepository, known port.SourceQueries, v domain.Vault) (source.Extract, error) {
	derived, err := c.DerivedStores().Open(v)
	if err != nil {
		return source.Extract{}, err
	}
	extract := source.NewExtract(c.VaultReaders(), sources, known)
	extract.Derived = derived
	extract.Documents = c.TextExtractor()
	extract.Sizes = c.Chunking()
	extract.Legibility = c.Legibility()
	extract.RebuildIndex = c.RebuildIndex
	return extract, nil
}
