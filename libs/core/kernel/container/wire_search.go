package container

import (
	"context"
	"errors"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostsearch"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	machinedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
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
// notes read, the file texts cut, and the vectors of the search source made.
func ReadWholeVault(c Config,
	ctx context.Context, db *Index, source searchsourceport.Source, v domain.Vault,
) (indexing.ReadWholeVault, error) {
	return readWholeVault(searchOf(c), ctx, db, source, v, makeRefresh(c, db, db.Notes()))
}

// Embed gives a vault's chunks the vectors of the search source they need.
func Embed(c Config, db *Index, source searchsourceport.Source, v domain.Vault) (vectorscase.Embed, error) {
	return searchEmbed(searchOf(c), db, source, v)
}

// IndexTexts cuts the text versions of a vault's files into chunks.
func IndexTexts(c Config, db *Index, v domain.Vault) (filetext.Index, error) {
	return indexTexts(searchOf(c), db, v, nil)
}

// NewSearch is the search a question is answered by, put together the one way.
func NewSearch(c Config, db *Index, asking searchsourceport.Source, errorHandler func(error)) searchusecase.Search {
	return newSearch(searchOf(c), db, asking, errorHandler)
}

// coreFolder is the folder of a vault's store that holds the core's own files.
const coreFolder = "core"

// searchWiring wires search, search policy, vectors and chunking of an installation.
type searchWiring struct {
	ServiceDir         string
	ShouldRebuildIndex bool
	Chunks             *note.ChunkTokens
	ResolvedCompute    machinedomain.ResolvedCompute
	PluginHost         *PluginHost
	RunningHost        *PluginHost
	Clock              machineport.Clock
	VaultReaders       vaultport.VaultReaders
	TextReaders        textdomain.Readers
	ErrorHandler       func(error)
}

func cutNotes(c searchWiring) chunking.CutNotes {
	cutter := chunking.NewCutNotes(c.Chunks.Get())
	cutter.Profiles = func(t domain.NoteType) (domain.NoteProfile, bool) { return c.PluginHost.Runtime.ResolveNoteProfile(t) }
	return cutter
}

func setNoteCut(c searchWiring, scan *vault.Scan) {
	scan.Cut = cutNotes(c)
	scan.ShouldRecut = c.Chunks.Get() > 0
}

func searchScan(c searchWiring, db *Index) vault.Scan {
	scan := vault.NewScan(
		c.VaultReaders,
		db.Vaults(),
		db.Notes(),
		db.NoteQueries(),
		db.Maintenance(),
		db.Sources(),
	)
	setNoteCut(c, &scan)
	scan.Walks = &db.Walks
	scan.ShouldRebuildIndex = c.ShouldRebuildIndex
	return scan
}

func readWholeVault(c searchWiring,
	ctx context.Context, db *Index, source searchsourceport.Source, v domain.Vault, refresh vault.Refresh,
) (indexing.ReadWholeVault, error) {
	if err := fitIndex(c, db).Execute(ctx, source); err != nil {
		return indexing.ReadWholeVault{}, err
	}
	return buildReadWholeVault(c, db, source, v, refresh)
}

func buildReadWholeVault(c searchWiring,
	db *Index, source searchsourceport.Source, v domain.Vault, refresh vault.Refresh,
) (indexing.ReadWholeVault, error) {
	refresh.Force = true
	texts, err := indexTexts(c, db, v, file.NewReadNoteAgain(createUnreadableFinder(refresh)).Execute)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	embed, err := searchEmbed(c, db, source, v)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	return indexing.NewReadWholeVault(searchScan(c, db), texts, embed), nil
}

func createUnreadableFinder(refresh vault.Refresh) func(ctx context.Context, v domain.Vault, paths []string) ([]string, error) {
	return func(ctx context.Context, v domain.Vault, paths []string) ([]string, error) {
		result, err := refresh.Execute(ctx, v, paths)
		return result.Unreadable, err
	}
}

func fitIndex(c searchWiring, db *Index) vectorscase.FitIndex {
	fit := vectorscase.NewFitIndex(db, c.ErrorHandler)
	fit.ShouldRebuildIndex = c.ShouldRebuildIndex
	return fit
}

func searchEmbed(c searchWiring, db *Index, source searchsourceport.Source, v domain.Vault) (vectorscase.Embed, error) {
	store, err := derivedfile.OpenDerived(v.Path, vaultconfig.Options{ServiceDir: c.ServiceDir}, coreFolder)
	if err != nil {
		return vectorscase.Embed{}, err
	}
	embed := vectorscase.NewEmbed(c.VaultReaders, db.Sources(), db.Sources())
	embed.Claim = store.NewIndexClaim(coreFolder + "/indexing")
	embed.Texts = c.TextReaders
	embed.Source = source
	return embed, nil
}

func indexTexts(
	c searchWiring, db *Index, v domain.Vault, readNote func(ctx context.Context, v domain.Vault, path string) error,
) (filetext.Index, error) {
	if _, err := c.VaultReaders.Open(v); err != nil {
		return filetext.Index{}, err
	}
	index := filetext.NewHandlingIndex(filetext.NewIndex(c.VaultReaders, db.Sources(), db.Sources(), db.Sources(), db.Sources(), c.TextReaders), c.ErrorHandler, readNote)
	cutter := cutNotes(c)
	index.Sizes = cutter.Sizes
	index.Legibility = cutter.Legibility
	index.ShouldRebuildIndex = c.ShouldRebuildIndex
	return index, nil
}

func newSearch(c searchWiring, db *Index, asking searchsourceport.Source, errorHandler func(error)) searchusecase.Search {
	return newSearchOver(c, db.ChunkSearch(), getParticipants(db, asking), errorHandler)
}

func getParticipants(db *Index, source searchsourceport.Source) []searchusecase.Participant {
	if source == nil {
		return nil
	}
	return []searchusecase.Participant{searchusecase.NewSource(source, db.ChunkSearch())}
}

func newSearchOver(c searchWiring, passages sourceport.PassageQueries, sources []searchusecase.Participant, errorHandler func(error)) searchusecase.Search {
	searching := searchusecase.New(passages, c.VaultReaders, c.TextReaders, sources, errorHandler)
	if workers := c.ResolvedCompute.Workers; workers > 0 {
		searching.Extractors = workers
	}
	return searching
}

func newFloorPolicy(db *Index, source searchsourceport.Source, errorHandler func(error)) searchusecase.FloorPolicy {
	return searchusecase.NewFloorPolicy(db.SearchFloors(), getFloorHint(source), errorHandler)
}

func setPluginSearch(c searchWiring, db *Index, source searchsourceport.Source, errorHandler func(error)) {
	access := hostsearch.Search{
		Samples:  db.SearchSamples(),
		Nearest:  searchusecase.NewGetBestSimilarity(db.ChunkSearch()),
		Places:   searchusecase.NewFindNearest(db.ChunkSearch()),
		Progress: db.ChunkVectors(),
		Stats:    searchusecase.NewStats(db.ChunkVectors(), c.Clock),
		Policy:   newFloorPolicy(db, source, errorHandler),
		Passages: searchusecase.NewSearchPassages(newSearch(c, db, source, errorHandler)),
	}
	if source != nil {
		access.Source = source
		access.Queries = newEmbedQueries(source)
	}
	c.PluginHost.Endpoint.SetPluginSearch(access.Access())
}

func searchSources(c searchWiring, ctx context.Context, tasks *run.Tasks) (indexing, asking searchsourceport.WatchedSource, closer func() error, why error) {
	source, closeSource, err := searchOpenSource(c, ctx, run.NewModelReadReporter(tasks))
	if err != nil {
		return nil, nil, func() error { return nil }, err
	}
	return source, source, closeSource, nil
}

func searchSource(c searchWiring, ctx context.Context) (searchsourceport.WatchedSource, func() error, error) {
	return searchOpenSource(c, ctx, nil)
}

func searchOpenSource(c searchWiring, ctx context.Context, onReadError func(error)) (searchsourceport.WatchedSource, func() error, error) {
	if _, err := c.RunningHost.Runtime.ResolveCapability(bridge.CapabilitySearchSource); errors.Is(err, domain.ErrUnavailable) {
		return nil, func() error { return nil }, nil
	} else if err != nil {
		return nil, nil, err
	}
	return openSource(c, ctx, c.RunningHost, onReadError)
}

func openSource(c searchWiring, ctx context.Context, host *PluginHost, onReadError func(error)) (searchsourceport.WatchedSource, func() error, error) {
	opened, err := sourcebridge.Open(ctx, host.Runtime)
	if err != nil {
		return nil, nil, err
	}
	if onReadError != nil {
		opened.OnDescriptionReadError(onReadError)
	}
	c.Chunks.Follow(opened, opened.OnDescriptionChange)
	return opened, opened.Close, nil
}

func newEmbedQueries(source searchsourceport.Source) *vectorscase.EmbedQueries {
	queries, ok := source.(searchsourceport.QueryEmbedder)
	if !ok {
		return nil
	}
	return vectorscase.NewEmbedQueries(queries)
}

func getFloorHint(source searchsourceport.Source) searchusecase.FloorHint {
	if advisor, ok := source.(searchsourceport.FloorAdvisor); ok {
		return advisor.GetFloorHint
	}
	return nil
}
