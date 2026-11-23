package container

import (
	"cmp"
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/fitindex"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// coreFolder is the folder of a vault's store that holds the core's own files.
const coreFolder = "core"

type noteChunkPolicy struct {
	Chunks   *note.ChunkTokens
	Profiles pluginResolver
}

type vaultIndexPlan struct {
	ChunkPolicy        noteChunkPolicy
	ServiceDir         string
	ShouldRebuildIndex bool
	VaultReaders       vaultport.VaultReaders
	TextReaders        textport.Readers
	HandleError        func(error)
}

func newTextSearch(plan vaultIndexPlan, workers int) textSearch {
	return textSearch{VaultReaders: plan.VaultReaders, TextReaders: plan.TextReaders, Workers: workers}
}

func cutNotes(c noteChunkPolicy) chunking.CutNotes {
	cutter := chunking.NewCutNotes(c.Chunks.Get())
	if c.Profiles != nil {
		cutter.Profiles = c.Profiles.ResolveNoteProfile
	}
	return cutter
}

func setNoteCut(c noteChunkPolicy, scan *indexing.Scan) {
	scan.Cut = cutNotes(c)
	scan.ShouldRecut = c.Chunks.Get() > 0
}

func searchScan(c vaultIndexPlan, db *vaultIndex) indexing.Scan {
	scan := indexing.NewScan(
		c.VaultReaders,
		db.Vaults(),
		db.Notes(),
		db.NoteQueries(),
		db.Maintenance(),
		db.Sources(),
	)
	setNoteCut(c.ChunkPolicy, &scan)
	scan.Walks = &db.Walks
	return scan.Rebuilding(c.ShouldRebuildIndex)
}

func readWholeVault(c vaultIndexPlan,
	ctx context.Context, db *vaultIndex, source searchsourceport.Source, v domain.Vault, refresh indexing.Refresh,
) (indexing.ReadWholeVault, error) {
	fit := fitindex.New(db.VectorIndex(), c.HandleError, c.ShouldRebuildIndex)
	if err := fit.Execute(ctx, source); err != nil {
		return indexing.ReadWholeVault{}, err
	}
	return buildReadWholeVault(c, db, source, v, refresh)
}

func buildReadWholeVault(c vaultIndexPlan,
	db *vaultIndex, source searchsourceport.Source, v domain.Vault, refresh indexing.Refresh,
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

func createUnreadableFinder(refresh indexing.Refresh) func(ctx context.Context, v domain.Vault, paths []string) ([]string, error) {
	return func(ctx context.Context, v domain.Vault, paths []string) ([]string, error) {
		result, err := refresh.Execute(ctx, v, paths)
		return result.Unreadable, err
	}
}

func searchEmbed(c vaultIndexPlan, db *vaultIndex, source searchsourceport.Source, v domain.Vault) (indexing.Embed, error) {
	store, err := derivedfile.OpenDerived(v.Path, vaultconfig.Options{ServiceDir: c.ServiceDir}, coreFolder)
	if err != nil {
		return indexing.Embed{}, err
	}
	return indexing.NewEmbed(
		c.VaultReaders, db.Sources(), db.Sources(), store.NewIndexClaim(coreFolder+"/indexing"), c.TextReaders, cmp.Or[searchsourceport.Source](source, none.SearchSource{}),
	), nil
}

func indexTexts(
	c vaultIndexPlan, db *vaultIndex, v domain.Vault, readNote func(ctx context.Context, v domain.Vault, path string) error,
) (indexing.Index, error) {
	if _, err := c.VaultReaders.Open(v); err != nil {
		return indexing.Index{}, err
	}
	cutter := cutNotes(c.ChunkPolicy)
	index := indexing.NewIndex(indexing.IndexDeps{
		Readers: c.VaultReaders, Sources: db.Sources(), Records: db.Sources(), Backlog: db.Sources(), Lookup: db.Sources(),
		Texts: c.TextReaders, Sizes: cutter.Sizes, Legibility: cutter.Legibility, ShouldRebuildIndex: c.ShouldRebuildIndex,
	})
	return indexing.NewHandlingIndex(index, c.HandleError, readNote), nil
}
