package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// GetChunkSizes is how a source's text is cut into chunks.
//
// It is a fact about the settings and not about what is loaded: the sizes are
// part of what a chunk is kept under, and a chunk cut one way in a window and
// another way in a terminal is a source cut again every time the two take
// turns. This is where the number comes from, and there is nowhere else to
// take it from.
func (c Config) GetChunkSizes() chunking.Sizes {
	if c.Embedding.Indexing.Use == "" {
		return chunking.Sizes{}
	}
	return chunking.Sizes{Limit: chunking.GetCharacterBound(c.Embedding.Model.MaxTokens)}
}

// Legibility is what a chunk has to read like to be indexed. It comes from here
// for the reason the sizes do: one answer for one installation.
func (c Config) Legibility() chunking.Legibility {
	return chunking.Legibility{}
}

// CutNotes is how a note is cut into the chunks the index keeps: at this
// installation's sizes, keeping what reads as text.
func (c Config) CutNotes() note.CutNotes {
	return note.CutNotes{Sizes: c.GetChunkSizes(), Legibility: c.Legibility()}
}

// Scan is the walk that reads a vault's notes into the index, cut at this
// installation's sizes. Every application that reads a vault takes it from here.
func (c Config) Scan(db *Index) vault.Scan {
	scan := vault.NewScan(
		c.VaultReaders(),
		db.Vaults(),
		db.Notes(),
		db.Queries(),
		db.Maintenance(),
	)
	scan.Cut = c.CutNotes()
	scan.Walks = db.Walks()
	scan.ShouldRebuildIndex = c.ShouldRebuildIndex
	return scan
}

// ReadWholeVault is what makes a vault answer, put together the one way: the
// notes read, the books read, and the vectors made. Every entry point takes it
// from here, so a vault made searchable in a terminal and a vault made
// searchable in a window are the same vault.
func (c Config) ReadWholeVault(
	ctx context.Context, db *Index, embedder port.Embedder, v domain.Vault,
) (vault.ReadWholeVault, error) {
	if embedder != nil {
		if err := c.syncVectorIndex(ctx, db, embedder.GetModel()); err != nil {
			return vault.ReadWholeVault{}, err
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

// syncVectorIndex fits the vector index to the model's width, and on a rebuild
// drops the vectors of other recipes.
func (c Config) syncVectorIndex(ctx context.Context, db *Index, model port.EmbeddingModel) error {
	if err := db.FitVectors(ctx, model.Dimensions, model.Recipe()); err != nil {
		return err
	}
	if !c.ShouldRebuildIndex {
		return nil
	}
	if _, err := db.DeleteVectorsOfOtherRecipes(ctx, model.Recipe()); err != nil {
		return err
	}
	// A failed compaction is reported and the rows stay deleted.
	if err := db.Compact(ctx); err != nil {
		c.handleError(err)
	}
	return nil
}

// Embed gives a vault's chunks the vectors they owe.
func (c Config) Embed(db *Index, embedder port.Embedder, v domain.Vault) (source.Embed, error) {
	derived, err := c.GetDerivedStores().Open(v)
	if err != nil {
		return source.Embed{}, err
	}
	embed := source.NewEmbed(c.VaultReaders(), db.VectorQueries(), db.Vectors())
	embed.Derived = derived
	embed.Documents = c.TextExtractor()
	embed.Embedder = embedder
	return embed, nil
}

// Extract cuts a vault's sources into chunks. Every entry point takes it from
// here, so what a chunk is kept under is one answer.
func (c Config) Extract(sources port.SourceRepository, known port.SourceQueries, v domain.Vault) (source.Extract, error) {
	derived, err := c.GetDerivedStores().Open(v)
	if err != nil {
		return source.Extract{}, err
	}
	extract := source.NewExtract(c.VaultReaders(), sources, known)
	extract.Derived = derived
	extract.Documents = c.TextExtractor()
	extract.Sizes = c.GetChunkSizes()
	extract.Legibility = c.Legibility()
	extract.ShouldRebuildIndex = c.ShouldRebuildIndex
	return extract, nil
}
