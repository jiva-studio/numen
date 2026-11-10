package container

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hostsearch"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// CutNotes is how a source's text and a note are cut into the chunks the index
// keeps: at this installation's sizes, keeping what reads as text.
func (c Config) CutNotes() query.CutNotes {
	return cutNotes(c.search())
}

// Scan is the walk that reads a vault's notes into the index, cut at this
// installation's sizes. Every application that reads a vault takes it from here.
func (c Config) Scan(db *vault.Index) vault.Scan {
	return searchScan(c.search(), db)
}

// ReadWholeVault is what makes a vault answer, put together the one way: the
// notes read, the file texts cut, and the vectors made.
func (c Config) ReadWholeVault(
	ctx context.Context, db *vault.Index, embedder embeddingport.Embedder, v domain.Vault,
) (vault.ReadWholeVault, error) {
	return readWholeVault(c.search(), ctx, db, embedder, v, makeRefresh(c, db, db.Notes()))
}

// FitIndex sets the vector index to the model of the embedding provider.
func (c Config) FitIndex(db *vault.Index) vectorscase.FitIndex {
	return fitIndex(c.search(), db)
}

// Embed gives a vault's chunks the vectors they need.
func (c Config) Embed(db *vault.Index, embedder embeddingport.Embedder, v domain.Vault) (vectorscase.Embed, error) {
	return searchEmbed(c.search(), db, embedder, v)
}

// IndexTexts cuts the text versions of a vault's files into chunks.
func (c Config) IndexTexts(db *vault.Index, v domain.Vault) (filetext.Index, error) {
	return indexTexts(c.search(), db, v)
}

// NewSearch is the search a question is answered by, put together the one way.
func (c Config) NewSearch(db *vault.Index, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	return newSearch(c.search(), db, asking, errorHandler)
}

// NewSearchOver is that search over the passages given, for a run that holds
// the index open for asking alone.
func (c Config) NewSearchOver(passages sourceport.PassageQueries, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	return newSearchOver(c.search(), passages, asking, errorHandler)
}

// NewFloorPolicy is the floor a search by meaning applies under the model of an embedder.
func (c Config) NewFloorPolicy(db *vault.Index, embedder embeddingport.Embedder, errorHandler func(error)) searchusecase.FloorPolicy {
	return newFloorPolicy(c.search(), db, embedder, errorHandler)
}

// SetPluginSearch lends the plugins the search index, the model that fills it and the floors of the
// vaults.
func (c Config) SetPluginSearch(db *vault.Index, embedder embeddingport.Embedder, errorHandler func(error)) {
	setPluginSearch(c.search(), db, embedder, errorHandler)
}

// EmbedderReady reports whether an embedding plugin is enabled.
func (c Config) EmbedderReady() bool {
	return embedderReady(c.search())
}

// Embedders returns the embedder that produces the vectors a vault is searched
// by and the embedder that produces the vector for a question.
func (c Config) Embedders(ctx context.Context, tasks *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error) {
	return searchEmbedders(c.search(), ctx, tasks)
}

// Embedder is the embedding plugin that produces the vectors of a vault.
func (c Config) Embedder(ctx context.Context) (embeddingport.Embedder, func() error, error) {
	return searchEmbedder(c.search(), ctx)
}

// coreFolder is the folder of a vault's store that holds the core's own files.
const coreFolder = "core"

// errNoteUnreadable is the error of a note read again whose file could not be read.
var errNoteUnreadable = errors.New("the note could not be read")

// searchWiring wires search, search policy, vectors and chunking of an installation.
type searchWiring struct {
	ServiceDir         string
	ShouldRebuildIndex bool
	Chunks             *note.ChunkTokens
	ResolvedCompute    hardware.ResolvedCompute
	PluginHost         *PluginHost
	Clock              machineport.Clock
	VaultReaders       vaultport.VaultReaders
	TextReaders        textport.TextReaders
	ErrorHandler       func(error)
	ResolvePlugin      func(capability string) (*PluginHost, error)
}

func cutNotes(c searchWiring) query.CutNotes {
	cutter := query.NewCutNotes(c.Chunks.Get())
	if c.PluginHost != nil {
		cutter.Profiles = c.PluginHost.ResolveNoteProfile
	}
	return cutter
}

func setNoteCut(c searchWiring, scan *vault.Scan, db *vault.Index) {
	scan.Cut = cutNotes(c)
	scan.ShouldRecut = c.Chunks.Get() > 0
	scan.GetOtherRecipePaths = func(ctx context.Context, v domain.VaultID, recipe string) ([]string, error) {
		recipes := []string{recipe, filetext.Index{Sizes: scan.Cut.Sizes}.Recipe()}
		return db.SourceBacklog().GetSourcesWithOtherRecipe(ctx, v, domain.KindNote, recipes, math.MaxInt32)
	}
}

func searchScan(c searchWiring, db *vault.Index) vault.Scan {
	scan := vault.NewScan(
		c.VaultReaders,
		db.Vaults(),
		db.Notes(),
		db.FileStats(),
		db.Maintenance(),
	)
	setNoteCut(c, &scan, db)
	scan.Walks = db.Walks()
	scan.ShouldRebuildIndex = c.ShouldRebuildIndex
	return scan
}

func readWholeVault(c searchWiring,
	ctx context.Context, db *vault.Index, embedder embeddingport.Embedder, v domain.Vault, refresh vault.Refresh,
) (vault.ReadWholeVault, error) {
	if err := fitIndex(c, db).Execute(ctx, embedder); err != nil {
		return vault.ReadWholeVault{}, err
	}
	texts, err := indexTexts(c, db, v)
	if err != nil {
		return vault.ReadWholeVault{}, err
	}
	refresh.Force = true
	texts.ReadNote = newNoteReader(refresh.Execute)
	embed, err := searchEmbed(c, db, embedder, v)
	if err != nil {
		return vault.ReadWholeVault{}, err
	}
	return vault.NewReadWholeVault(searchScan(c, db), texts, embed), nil
}

func newNoteReader(
	refresh func(ctx context.Context, v domain.Vault, paths []string) (vault.RefreshResult, error),
) func(ctx context.Context, v domain.Vault, path string) error {
	return func(ctx context.Context, v domain.Vault, path string) error {
		res, err := refresh(ctx, v, []string{path})
		if err != nil {
			return err
		}
		if slices.Contains(res.Unreadable, path) {
			return fmt.Errorf("%s: %w", path, errNoteUnreadable)
		}
		return nil
	}
}

func fitIndex(c searchWiring, db *vault.Index) vectorscase.FitIndex {
	fit := vectorscase.NewFitIndex(db)
	fit.ShouldRebuildIndex = c.ShouldRebuildIndex
	fit.ErrorHandler = c.ErrorHandler
	return fit
}

func searchEmbed(c searchWiring, db *vault.Index, embedder embeddingport.Embedder, v domain.Vault) (vectorscase.Embed, error) {
	store, err := derivedfile.OpenDerived(v.Path, vaultconfig.Options{ServiceDir: c.ServiceDir}, coreFolder)
	if err != nil {
		return vectorscase.Embed{}, err
	}
	embed := vectorscase.NewEmbed(c.VaultReaders, db.VectorQueries(), db.Vectors())
	embed.Claim = store.NewIndexClaim(coreFolder + "/indexing")
	embed.Texts = c.TextReaders
	embed.Embedder = embedder
	return embed, nil
}

func indexTexts(c searchWiring, db *vault.Index, v domain.Vault) (filetext.Index, error) {
	if _, err := c.VaultReaders.Open(v); err != nil {
		return filetext.Index{}, err
	}
	index := filetext.NewIndex(c.VaultReaders, db.Sources(), db.SourceRecords(), db.SourceBacklog(), db.RemovalLookup(), c.TextReaders)
	index.ErrorHandler = c.ErrorHandler
	cutter := cutNotes(c)
	index.Sizes = cutter.Sizes
	index.Legibility = cutter.Legibility
	index.ShouldRebuildIndex = c.ShouldRebuildIndex
	return index, nil
}

func newSearch(c searchWiring, db *vault.Index, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	policy := newFloorPolicy(c, db, asking, c.ErrorHandler)
	return newSearchOver(c, db.Passages(), asking, errorHandler).WithFloorSource(policy.GetFloor)
}

func newSearchOver(c searchWiring, passages sourceport.PassageQueries, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	policy := searchusecase.NewFloorPolicy(nil, getFloorHint(asking), nil)
	return searchusecase.New(passages, c.VaultReaders, c.TextReaders, asking, newEmbedQueries(asking), errorHandler).
		WithExtractors(c.ResolvedCompute.Workers).
		WithFloorSource(policy.GetFloor)
}

func newFloorPolicy(_ searchWiring, db *vault.Index, embedder embeddingport.Embedder, errorHandler func(error)) searchusecase.FloorPolicy {
	return searchusecase.NewFloorPolicy(db.SearchFloors(), getFloorHint(embedder), errorHandler)
}

func setPluginSearch(c searchWiring, db *vault.Index, embedder embeddingport.Embedder, errorHandler func(error)) {
	if c.PluginHost == nil {
		return
	}
	c.PluginHost.SetPluginSearch(hostsearch.Search{
		Embedder: embedder,
		Queries:  newEmbedQueries(embedder),
		Samples:  db.SearchSamples(),
		Nearest:  db.Nearest(),
		Progress: db.Progress(),
		Stats:    hostsearch.NewStats(db.Stats(), c.Clock),
		Policy:   newFloorPolicy(c, db, embedder, errorHandler),
		Passages: searchusecase.NewSearchPassages(newSearch(c, db, embedder, errorHandler)),
	}.Access())
}

func newEmbedQueries(embedder embeddingport.Embedder) *vectorscase.EmbedQueries {
	queries, ok := embedder.(embeddingport.QueryEmbedder)
	if !ok {
		return vectorscase.NewEmbedQueries(embedder, nil)
	}
	return vectorscase.NewEmbedQueries(embedder, queries)
}

func getFloorHint(embedder embeddingport.Embedder) searchusecase.FloorHint {
	advisor, ok := embedder.(embeddingport.FloorAdvisor)
	if !ok {
		return nil
	}
	return func(ctx context.Context) float64 { return advisor.GetFloorHint(ctx) }
}

func embedderReady(c searchWiring) bool {
	if c.ResolvePlugin == nil {
		return false
	}
	_, err := c.ResolvePlugin(bridge.CapabilityEmbedding)
	return err == nil
}

func searchEmbedders(c searchWiring, ctx context.Context, tasks *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error) {
	embedder, closeEmbedder, err := searchOpenEmbedder(c, ctx, newModelReadReporter(tasks))
	if err != nil {
		return nil, nil, func() error { return nil }, err
	}
	return embedder, embedder, closeEmbedder, nil
}

func searchEmbedder(c searchWiring, ctx context.Context) (embeddingport.Embedder, func() error, error) {
	return searchOpenEmbedder(c, ctx, nil)
}

// readingTheModel is the ID of the task that reports a read of the embedding model that failed.
const readingTheModel = "read embedding model"

func newModelReadReporter(tasks *run.Tasks) func(error) {
	if tasks == nil {
		return nil
	}
	return func(err error) {
		tasks.Set(run.Task{ID: readingTheModel, Title: "Read embedding model", Error: err.Error()})
	}
}

func searchOpenEmbedder(c searchWiring, ctx context.Context, onReadError func(error)) (embeddingport.Embedder, func() error, error) {
	if c.ResolvePlugin == nil {
		return nil, func() error { return nil }, nil
	}
	host, err := c.ResolvePlugin(bridge.CapabilityEmbedding)
	if errors.Is(err, domain.ErrUnavailable) {
		return nil, func() error { return nil }, nil
	}
	if err != nil {
		return nil, nil, err
	}
	opened, err := embedderbridge.Open(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	if onReadError != nil {
		opened.OnModelReadError(onReadError)
	}
	followChunkTokens(c.Chunks, opened, opened.OnModelChange)
	return opened, opened.Close, nil
}

func followChunkTokens(cell *note.ChunkTokens, embedder embeddingport.Embedder, onChange func(func())) {
	cell.Set(embedder.GetModel().ChunkTokens)
	onChange(func() { cell.Set(embedder.GetModel().ChunkTokens) })
}
