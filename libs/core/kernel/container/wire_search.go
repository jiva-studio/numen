package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	searchrunusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchrun"

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
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

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
	return scan.Rebuilding(c.ShouldRebuildIndex)
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
	return vectorscase.NewFitIndex(db.VectorIndex(), c.ErrorHandler, c.ShouldRebuildIndex)
}

func searchEmbed(c searchWiring, db *Index, source searchsourceport.Source, v domain.Vault) (vectorscase.Embed, error) {
	store, err := derivedfile.OpenDerived(v.Path, vaultconfig.Options{ServiceDir: c.ServiceDir}, coreFolder)
	if err != nil {
		return vectorscase.Embed{}, err
	}
	return vectorscase.NewEmbed(
		c.VaultReaders, db.Sources(), db.Sources(), store.NewIndexClaim(coreFolder+"/indexing"), c.TextReaders, getSearchSource(source),
	), nil
}

func indexTexts(
	c searchWiring, db *Index, v domain.Vault, readNote func(ctx context.Context, v domain.Vault, path string) error,
) (filetext.Index, error) {
	if _, err := c.VaultReaders.Open(v); err != nil {
		return filetext.Index{}, err
	}
	cutter := cutNotes(c)
	index := filetext.NewIndex(filetext.IndexDeps{
		Readers: c.VaultReaders, Sources: db.Sources(), Records: db.Sources(), Backlog: db.Sources(), Lookup: db.Sources(),
		Texts: c.TextReaders, Sizes: cutter.Sizes, Legibility: cutter.Legibility, ShouldRebuildIndex: c.ShouldRebuildIndex,
	})
	return filetext.NewHandlingIndex(index, c.ErrorHandler, readNote), nil
}

func newSearch(c searchWiring, db *Index, asking searchsourceport.Source, errorHandler func(error), reportAnswered func()) searchusecase.Search {
	return newSearchOver(c, db.ChunkSearch(), getParticipants(db, asking), errorHandler, reportAnswered)
}

func getParticipants(db *Index, source searchsourceport.Source) []searchusecase.Participant {
	if source == nil {
		return nil
	}
	return []searchusecase.Participant{searchusecase.NewSource(source, db.ChunkSearch())}
}

func newSearchOver(c searchWiring, queries sourceport.LexicalQueries, sources []searchusecase.Participant, errorHandler func(error), reportAnswered func()) searchusecase.Search {
	searching := searchusecase.New(queries, c.VaultReaders, c.TextReaders, sources, errorHandler, reportAnswered)
	if workers := c.ResolvedCompute.Workers; workers > 0 {
		searching.Extractors = workers
	}
	return searching
}

func newFloorPolicy(db *Index, source searchsourceport.Source, errorHandler func(error)) searchusecase.FloorPolicy {
	return searchusecase.NewFloorPolicy(db.SearchFloors(), getFloorHint(source), errorHandler)
}

func setPluginSearch(
	c searchWiring, db *Index, source searchsourceport.Source, errorHandler func(error), runs *searchrunusecase.SearchRuns,
) {
	policy := newFloorPolicy(db, source, errorHandler)
	access := hostsearch.Search{
		Samples:    db.SearchFloors(),
		Nearest:    searchusecase.NewGetBestSimilarity(db.ChunkSearch()),
		Places:     searchusecase.NewFindNearest(db.ChunkSearch()),
		Progress:   db.ChunkVectors(),
		Stats:      searchusecase.NewStats(db.ChunkVectors(), c.Clock),
		Policy:     policy,
		TextSearch: searchusecase.NewSearchText(newSearch(c, db, source, errorHandler, nil)),
		Source:     source,
		Queries:    newEmbedQueries(source),
	}
	c.PluginHost.Lender.Update(func(l *hostendpoint.Lent) {
		l.Search = access.Access(runs)
	})
}

func searchSources(c searchWiring, ctx context.Context) (indexing, asking searchsourceport.WatchedSource, closer func() error, why error) {
	report := pluginhealth.NewReportSearchSource(c.PluginHost.Health)
	source, closeSource, err := searchOpenSource(c, ctx, func(
		ctx context.Context, activator bridge.Activator, plugins sourcebridge.Plugins, kept *sourcebridge.Descriptions,
	) (*sourcebridge.Source, error) {
		found, err := plugins.ResolveCapability(bridge.CapabilitySearchSource)
		if err != nil {
			return nil, err
		}
		return sourcebridge.OpenWatching(ctx, activator, plugins, kept, func(readErr error) { report.Execute(found.Manifest.ID, readErr) })
	})
	if err != nil {
		return nil, nil, func() error { return nil }, err
	}
	return source, source, closeSource, nil
}

func searchSource(c searchWiring, ctx context.Context) (searchsourceport.WatchedSource, func() error, error) {
	return searchOpenSource(c, ctx, sourcebridge.Open)
}

// sourceOpener opens the search source over the plugin host, with the descriptions the host keeps.
type sourceOpener func(
	ctx context.Context, activator bridge.Activator, plugins sourcebridge.Plugins, kept *sourcebridge.Descriptions,
) (*sourcebridge.Source, error)

func searchOpenSource(c searchWiring, ctx context.Context, open sourceOpener) (searchsourceport.WatchedSource, func() error, error) {
	if _, err := c.RunningHost.Runtime.ResolveCapability(bridge.CapabilitySearchSource); errors.Is(err, domain.ErrUnavailable) {
		return nil, func() error { return nil }, nil
	} else if err != nil {
		return nil, nil, err
	}
	return openSource(c, ctx, c.RunningHost, open)
}

func openSource(c searchWiring, ctx context.Context, host *PluginHost, open sourceOpener) (searchsourceport.WatchedSource, func() error, error) {
	opened, err := open(ctx, host.Runtime, host.Runtime, host.SourceDescriptions)
	if err != nil {
		return nil, nil, err
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
