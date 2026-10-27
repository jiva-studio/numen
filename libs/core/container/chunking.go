package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
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
// notes read, the file texts cut, and the vectors made. Every entry point takes it
// from here, so a vault made searchable in a terminal and a vault made
// searchable in a window are the same vault.
func (c Config) ReadWholeVault(
	ctx context.Context, db *Index, embedder embeddingport.Embedder, v domain.Vault,
) (vault.ReadWholeVault, error) {
	if err := c.FitIndex(db).Execute(ctx, embedder); err != nil {
		return vault.ReadWholeVault{}, err
	}
	texts, err := c.IndexTexts(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		return vault.ReadWholeVault{}, err
	}
	embed, err := c.Embed(db, embedder, v)
	if err != nil {
		return vault.ReadWholeVault{}, err
	}
	return vault.NewReadWholeVault(c.Scan(db), texts, embed), nil
}

// FitIndex sets the vector index to the model of the embedding provider.
func (c Config) FitIndex(db *Index) vectors.FitIndex {
	fit := vectors.NewFitIndex(db)
	fit.ShouldRebuildIndex = c.ShouldRebuildIndex
	fit.ErrorHandler = c.handleError
	return fit
}

// Embed gives a vault's chunks the vectors they owe.
func (c Config) Embed(db *Index, embedder embeddingport.Embedder, v domain.Vault) (vectors.Embed, error) {
	store, err := derivedfile.OpenDerived(v.Path, c.vaultOptions(), coreFolder)
	if err != nil {
		return vectors.Embed{}, err
	}
	embed := vectors.NewEmbed(c.VaultReaders(), db.VectorQueries(), db.Vectors())
	embed.Claim = indexClaim{store: store}
	embed.Texts = c.getTextReaders()
	embed.Embedder = embedder
	return embed, nil
}

// IndexTexts cuts the text versions of a vault's files into chunks. Every entry
// point takes it from here, so what a chunk is kept under is one answer.
func (c Config) IndexTexts(sources sourceport.SourceRepository, queries sourceport.SourceQueries, v domain.Vault) (filetext.Index, error) {
	if _, err := c.VaultReaders().Open(v); err != nil {
		return filetext.Index{}, err
	}
	index := filetext.NewIndex(c.VaultReaders(), sources, queries, c.getTextReaders())
	index.ErrorHandler = c.handleError
	cutter := c.CutNotes()
	index.Sizes = cutter.Sizes
	index.Legibility = cutter.Legibility
	index.ShouldRebuildIndex = c.ShouldRebuildIndex
	return index, nil
}

// indexClaim is the hold on a vault's vectors: a claim on one name of the
// core's folder of that vault's store.
type indexClaim struct {
	store *derivedfile.DerivedStore
}

func (c indexClaim) ClaimIndex(ctx context.Context) (func() error, error) {
	return c.store.ClaimName(ctx, coreFolder+"/indexing")
}
