package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/source"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/usecase/vectors"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

// CutNotes is how a source's text and a note are cut into the chunks the index
// keeps: at this installation's sizes, keeping what reads as text.
func (c Config) CutNotes() note.CutNotes {
	cutter := note.NewCutNotes(c.Embedding.MaxTokens)
	if c.PluginHost != nil {
		cutter.Profiles = c.PluginHost.ResolveNoteProfile
	}
	return cutter
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
	ctx context.Context, db *Index, embedder embeddingport.Embedder, v domain.Vault,
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
	embed, err := c.Embed(db, embedder, v)
	if err != nil {
		return vault.ReadWholeVault{}, err
	}
	return vault.NewReadWholeVault(c.Scan(db), books, embed), nil
}

// syncVectorIndex fits the vector index to the model's width, and on a rebuild
// drops the vectors of other recipes.
func (c Config) syncVectorIndex(ctx context.Context, db *Index, model embeddingport.EmbeddingModel) error {
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
func (c Config) Embed(db *Index, embedder embeddingport.Embedder, v domain.Vault) (vectors.Embed, error) {
	derived, err := c.GetDerivedStores().Open(v)
	if err != nil {
		return vectors.Embed{}, err
	}
	embed := vectors.NewEmbed(c.VaultReaders(), db.VectorQueries(), db.Vectors())
	embed.Derived = derived
	embed.Texts = c.getTextReaders()
	embed.Embedder = embedder
	return embed, nil
}

// Extract cuts a vault's sources into chunks. Every entry point takes it from
// here, so what a chunk is kept under is one answer.
func (c Config) Extract(sources sourceport.SourceRepository, queries sourceport.SourceQueries, v domain.Vault) (source.Extract, error) {
	if _, err := c.VaultReaders().Open(v); err != nil {
		return source.Extract{}, err
	}
	extract := source.NewExtract(c.VaultReaders(), sources, queries, c.getTextReaders())
	extract.ErrorHandler = c.handleError
	cutter := c.CutNotes()
	extract.Sizes = cutter.Sizes
	extract.Legibility = cutter.Legibility
	extract.ShouldRebuildIndex = c.ShouldRebuildIndex
	return extract, nil
}
