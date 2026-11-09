package container

import (
	"context"

	searchwiring "github.com/jiva-studio/numen/libs/core/kernel/container/search"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ChunkTokens is the chunk size the embedding plugin reports, shared by every copy of Config.
type ChunkTokens = searchwiring.ChunkTokens

var errNoteUnreadable = searchwiring.ErrNoteUnreadable

func newNoteReader(
	refresh func(ctx context.Context, v domain.Vault, paths []string) (vault.RefreshResult, error),
) func(ctx context.Context, v domain.Vault, path string) error {
	return searchwiring.NewNoteReader(refresh)
}

func newModelReadReporter(tasks *run.Tasks) func(error) {
	return searchwiring.NewModelReadReporter(tasks)
}

// CutNotes is how a source's text and a note are cut into the chunks the index
// keeps: at this installation's sizes, keeping what reads as text.
func (c Config) CutNotes() note.CutNotes {
	return c.search().CutNotes()
}

// Scan is the walk that reads a vault's notes into the index, cut at this
// installation's sizes. Every application that reads a vault takes it from here.
func (c Config) Scan(db *Index) vault.Scan {
	return c.search().Scan(db)
}

// ReadWholeVault is what makes a vault answer, put together the one way: the
// notes read, the file texts cut, and the vectors made.
func (c Config) ReadWholeVault(
	ctx context.Context, db *Index, embedder embeddingport.Embedder, v domain.Vault,
) (vault.ReadWholeVault, error) {
	return c.search().ReadWholeVault(ctx, db, embedder, v, makeRefresh(c, db, db.Notes()))
}

// FitIndex sets the vector index to the model of the embedding provider.
func (c Config) FitIndex(db *Index) vectors.FitIndex {
	return c.search().FitIndex(db)
}

// Embed gives a vault's chunks the vectors they need.
func (c Config) Embed(db *Index, embedder embeddingport.Embedder, v domain.Vault) (vectors.Embed, error) {
	return c.search().Embed(db, embedder, v)
}

// IndexTexts cuts the text versions of a vault's files into chunks.
func (c Config) IndexTexts(sources sourceport.SourceRepository, queries sourceport.SourceQueries, v domain.Vault) (filetext.Index, error) {
	return c.search().IndexTexts(sources, queries, v)
}

// NewSearch is the search a question is answered by, put together the one way.
func (c Config) NewSearch(db *Index, asking embeddingport.Embedder, errorHandler func(error)) search.Search {
	return c.search().NewSearch(db, asking, errorHandler)
}

// NewSearchOver is that search over the passages given, for a run that holds
// the index open for asking alone.
func (c Config) NewSearchOver(passages sourceport.PassageQueries, asking embeddingport.Embedder, errorHandler func(error)) search.Search {
	return c.search().NewSearchOver(passages, asking, errorHandler)
}

// NewFloorPolicy is the floor a search by meaning applies under the model of an embedder.
func (c Config) NewFloorPolicy(db *Index, embedder embeddingport.Embedder, errorHandler func(error)) search.FloorPolicy {
	return c.search().NewFloorPolicy(db, embedder, errorHandler)
}

// SetPluginSearch lends the plugins the search index, the model that fills it and the floors of the
// vaults.
func (c Config) SetPluginSearch(db *Index, embedder embeddingport.Embedder, errorHandler func(error)) {
	c.search().SetPluginSearch(db, embedder, errorHandler)
}

// EmbedderReady reports whether an embedding plugin is enabled.
func (c Config) EmbedderReady() bool {
	return c.search().EmbedderReady()
}

// Embedders returns the embedder that produces the vectors a vault is searched
// by and the embedder that produces the vector for a question.
func (c Config) Embedders(ctx context.Context, tasks *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error) {
	return c.search().Embedders(ctx, tasks)
}

// Embedder is the embedding plugin that produces the vectors of a vault.
func (c Config) Embedder(ctx context.Context) (embeddingport.Embedder, func() error, error) {
	return c.search().Embedder(ctx)
}
