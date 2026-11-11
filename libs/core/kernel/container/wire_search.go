package container

import (
	"context"
	"errors"
	"math"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostsearch"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Scan is the walk that reads a vault's notes into the index, cut at this
// installation's sizes. Every application that reads a vault takes it from here.
func Scan(c Config, db *Index) vault.Scan {
	return searchScan(searchOf(c), db)
}

// ReadWholeVault is what makes a vault answer, put together the one way: the
// notes read, the file texts cut, and the vectors made.
func ReadWholeVault(c Config,
	ctx context.Context, db *Index, embedder embeddingport.Embedder, v domain.Vault,
) (indexing.ReadWholeVault, error) {
	return readWholeVault(searchOf(c), ctx, db, embedder, v, makeRefresh(c, db, db.Notes()))
}

// Embed gives a vault's chunks the vectors they need.
func Embed(c Config, db *Index, embedder embeddingport.Embedder, v domain.Vault) (vectorscase.Embed, error) {
	return searchEmbed(searchOf(c), db, embedder, v)
}

// IndexTexts cuts the text versions of a vault's files into chunks.
func IndexTexts(c Config, db *Index, v domain.Vault) (filetext.Index, error) {
	return indexTexts(searchOf(c), db, v)
}

// NewSearch is the search a question is answered by, put together the one way.
func NewSearch(c Config, db *Index, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	return newSearch(searchOf(c), db, asking, errorHandler)
}

// coreFolder is the folder of a vault's store that holds the core's own files.
const coreFolder = "core"

// searchWiring wires search, search policy, vectors and chunking of an installation.
type searchWiring struct {
	ServiceDir         string
	ShouldRebuildIndex bool
	Chunks             *note.ChunkTokens
	ResolvedCompute    hardware.ResolvedCompute
	PluginHost         *PluginHost
	Clock              machineport.Clock
	VaultReaders       vaultport.VaultReaders
	TextReaders        textdomain.Readers
	ErrorHandler       func(error)
	ResolvePlugin      func(capability string) (*PluginHost, error)
}

func cutNotes(c searchWiring) chunking.CutNotes {
	cutter := chunking.NewCutNotes(c.Chunks.Get())
	cutter.Profiles = c.PluginHost.Runtime.ResolveNoteProfile
	return cutter
}

func setNoteCut(c searchWiring, scan *vault.Scan, db *Index) {
	scan.Cut = cutNotes(c)
	scan.ShouldRecut = c.Chunks.Get() > 0
	scan.GetOtherRecipePaths = func(ctx context.Context, v domain.VaultID, recipe string) ([]string, error) {
		recipes := []string{recipe, filetext.Index{Sizes: scan.Cut.Sizes}.Recipe()}
		return db.Sources().GetSourcesWithOtherRecipe(ctx, v, domain.KindNote, recipes, math.MaxInt32)
	}
}

func searchScan(c searchWiring, db *Index) vault.Scan {
	scan := vault.NewScan(
		c.VaultReaders,
		db.Vaults(),
		db.Notes(),
		db.NoteQueries(),
		db.Maintenance(),
	)
	setNoteCut(c, &scan, db)
	scan.Walks = &db.Walks
	scan.ShouldRebuildIndex = c.ShouldRebuildIndex
	return scan
}

func readWholeVault(c searchWiring,
	ctx context.Context, db *Index, embedder embeddingport.Embedder, v domain.Vault, refresh vault.Refresh,
) (indexing.ReadWholeVault, error) {
	if err := fitIndex(c, db).Execute(ctx, embedder); err != nil {
		return indexing.ReadWholeVault{}, err
	}
	return buildReadWholeVault(c, db, embedder, v, refresh)
}

func buildReadWholeVault(c searchWiring,
	db *Index, embedder embeddingport.Embedder, v domain.Vault, refresh vault.Refresh,
) (indexing.ReadWholeVault, error) {
	texts, err := indexTexts(c, db, v)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	refresh.Force = true
	texts.ReadNote = file.NewReadNoteAgain(getUnreadable(refresh)).Execute
	embed, err := searchEmbed(c, db, embedder, v)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	return indexing.NewReadWholeVault(searchScan(c, db), texts, embed), nil
}

func getUnreadable(refresh vault.Refresh) func(ctx context.Context, v domain.Vault, paths []string) ([]string, error) {
	return func(ctx context.Context, v domain.Vault, paths []string) ([]string, error) {
		result, err := refresh.Execute(ctx, v, paths)
		return result.Unreadable, err
	}
}

func fitIndex(c searchWiring, db *Index) vectorscase.FitIndex {
	fit := vectorscase.NewFitIndex(db)
	fit.ShouldRebuildIndex = c.ShouldRebuildIndex
	fit.ErrorHandler = c.ErrorHandler
	return fit
}

func searchEmbed(c searchWiring, db *Index, embedder embeddingport.Embedder, v domain.Vault) (vectorscase.Embed, error) {
	store, err := derivedfile.OpenDerived(v.Path, vaultconfig.Options{ServiceDir: c.ServiceDir}, coreFolder)
	if err != nil {
		return vectorscase.Embed{}, err
	}
	embed := vectorscase.NewEmbed(c.VaultReaders, db.Sources(), db.Sources())
	embed.Claim = store.NewIndexClaim(coreFolder + "/indexing")
	embed.Texts = c.TextReaders
	embed.Embedder = embedder
	return embed, nil
}

func indexTexts(c searchWiring, db *Index, v domain.Vault) (filetext.Index, error) {
	if _, err := c.VaultReaders.Open(v); err != nil {
		return filetext.Index{}, err
	}
	index := filetext.NewIndex(c.VaultReaders, db.Sources(), db.Sources(), db.Sources(), db.Sources(), c.TextReaders)
	index.ErrorHandler = c.ErrorHandler
	cutter := cutNotes(c)
	index.Sizes = cutter.Sizes
	index.Legibility = cutter.Legibility
	index.ShouldRebuildIndex = c.ShouldRebuildIndex
	return index, nil
}

func newSearch(c searchWiring, db *Index, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	policy := newFloorPolicy(c, db, asking, c.ErrorHandler)
	searching := newSearchOver(c, db.ChunkSearch(), asking, errorHandler)
	searching.Floor = policy.GetFloor
	return searching
}

func newSearchOver(c searchWiring, passages sourceport.PassageQueries, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	policy := searchusecase.NewFloorPolicy(nil, getFloorHint(asking), nil)
	searching := searchusecase.New(passages, c.VaultReaders, c.TextReaders, asking, newEmbedQueries(asking), errorHandler)
	if workers := c.ResolvedCompute.Workers; workers > 0 {
		searching.Extractors = workers
	}
	searching.Floor = policy.GetFloor
	return searching
}

func newFloorPolicy(_ searchWiring, db *Index, embedder embeddingport.Embedder, errorHandler func(error)) searchusecase.FloorPolicy {
	return searchusecase.NewFloorPolicy(db.SearchFloors(), getFloorHint(embedder), errorHandler)
}

func setPluginSearch(c searchWiring, db *Index, embedder embeddingport.Embedder, errorHandler func(error)) {
	c.PluginHost.Endpoint.SetPluginSearch(hostsearch.Search{
		Embedder: embedder,
		Queries:  newEmbedQueries(embedder),
		Samples:  db.SearchSamples(),
		Nearest:  db.ChunkSearch(),
		Progress: db.ChunkVectors(),
		Stats:    hostsearch.NewStats(db.ChunkVectors(), c.Clock),
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
		tasks.Set(work.Task{ID: readingTheModel, Title: "Read embedding model", Error: err.Error()})
	}
}

func searchOpenEmbedder(c searchWiring, ctx context.Context, onReadError func(error)) (embeddingport.Embedder, func() error, error) {
	host, err := c.ResolvePlugin(bridge.CapabilityEmbedding)
	if errors.Is(err, domain.ErrUnavailable) {
		return nil, func() error { return nil }, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return openEmbedder(c, ctx, host, onReadError)
}

func openEmbedder(c searchWiring, ctx context.Context, host *PluginHost, onReadError func(error)) (embeddingport.Embedder, func() error, error) {
	opened, err := embedderbridge.Open(ctx, host.Runtime)
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
