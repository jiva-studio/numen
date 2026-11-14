package editor

import (
	"context"
	"fmt"
	"io"
	"slices"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"

	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	indexnote "github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	"github.com/jiva-studio/numen/libs/core/kernel/port/indexport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// derivedCoreFolder is the folder of a vault's store that holds the core's own files.
const derivedCoreFolder = "core"

// readingParts is the parts of an installation that read a vault into a cache at a path. The
// composition root imports this package, so its tests assemble them here.
type readingParts struct {
	indexPath          string
	shouldRebuildIndex bool
	texts              textdomain.Readers
	chunks             *note.ChunkTokens
	watches            *vaultwatch.Hub
}

func newReadingParts(indexPath string, shouldRebuildIndex bool, texts textdomain.Readers) readingParts {
	return readingParts{
		indexPath:          indexPath,
		shouldRebuildIndex: shouldRebuildIndex,
		texts:              texts,
		chunks:             &note.ChunkTokens{},
		watches:            vaultwatch.NewHub(),
	}
}

// OpenIndex opens the cache. The caller closes it.
func (w readingParts) OpenIndex(ctx context.Context) (*testIndex, error) {
	db, err := index.Open(ctx, w.indexPath)
	if err != nil {
		return nil, err
	}
	queries := db.NoteQueries()
	target := &testIndex{closer: db, vectors: db.VectorIndex(), ledger: vault.NewLedger()}
	target.Stores.Vaults, target.Stores.Notes, target.Stores.Maintenance = db.Vaults(), db.Notes(), db.Maintenance()
	target.NoteQueries.FileStats, target.NoteQueries.Headings, target.NoteQueries.Links = queries, queries, queries
	target.NoteQueries.LinkBatches = queries
	target.NoteQueries.NamedPaths, target.NoteQueries.NoteRefs, target.NoteQueries.NoteTypes = queries, queries, queries
	target.NoteQueries.OpeningNote = queries
	target.Chunks.Passages, target.Chunks.Nearest, target.Chunks.Progress = db.ChunkSearch(), db.ChunkSearch(), db.ChunkVectors()
	target.Chunks.Vectors, target.Chunks.VectorQueries = db.Sources(), db.Sources()
	target.Source.Sources, target.Source.SourceRecords, target.Source.SourceBacklog = db.Sources(), db.Sources(), db.Sources()
	target.Source.RemovalLookup = db.Sources()
	return target, nil
}

// testIndex is the part of an open index the tests of the window read and write.
type testIndex struct {
	closer  io.Closer
	vectors searchsourceport.VectorIndex
	walks   vault.Walks
	ledger  *vault.Ledger

	Stores struct {
		Vaults      vaultport.VaultRepository
		Notes       noteport.NoteRepository
		Maintenance indexport.IndexMaintenance
	}
	NoteQueries struct {
		FileStats   noteport.FileStats
		Headings    noteport.Headings
		Links       noteport.LinkQueries
		LinkBatches noteport.LinkBatches
		NamedPaths  noteport.NamedPaths
		NoteRefs    noteport.NoteRefs
		NoteTypes   noteport.NoteTypes
		OpeningNote noteport.OpeningNote
	}
	Chunks struct {
		Passages      sourceport.PassageQueries
		Nearest       sourceport.NearestQueries
		Progress      indexport.IndexProgress
		Vectors       searchsourceport.VectorRepository
		VectorQueries searchsourceport.VectorQueries
	}
	Source struct {
		Sources       sourceport.SourceRepository
		SourceRecords sourceport.SourceRecords
		SourceBacklog sourceport.SourceBacklog
		RemovalLookup sourceport.RemovalLookup
	}
}

func (i *testIndex) Close() error { return i.closer.Close() }

func (i *testIndex) Walks() *vault.Walks { return &i.walks }

func (i *testIndex) Ledger() *vault.Ledger { return i.ledger }

func (i *testIndex) VectorIndex() searchsourceport.VectorIndex { return i.vectors }

func (w readingParts) GetVaultReaders() vaultport.VaultReaders { return vaultreader.Readers{} }

// GetVaultWatcher monitors vaults for external filesystem changes.
func (w readingParts) GetVaultWatcher() vaultport.VaultWatcher {
	return vaultwatch.Watcher{Hub: w.watches.GetShared()}
}

func (w readingParts) cutNotes() chunking.CutNotes {
	return chunking.NewCutNotes(w.chunks.Get())
}

func (w readingParts) setNoteCut(scan *vault.Scan) {
	scan.Cut = w.cutNotes()
	scan.ShouldRecut = w.chunks.Get() > 0
}

func (w readingParts) scan(db *testIndex, rebuild bool) vault.Scan {
	scan := vault.NewScan(w.GetVaultReaders(), db.Stores.Vaults, db.Stores.Notes, db.NoteQueries.FileStats, db.Stores.Maintenance, db.Source.SourceBacklog)
	w.setNoteCut(&scan)
	scan.Walks = db.Walks()
	scan.ShouldRebuildIndex = rebuild
	return scan
}

// GetRefresh brings named notes up to date in the index, over the notes of the cache.
func (w readingParts) GetRefresh(db *testIndex) vault.Refresh {
	return w.makeRefresh(db, db.Stores.Notes)
}

func (w readingParts) makeRefresh(db *testIndex, notes noteport.NoteRepository) vault.Refresh {
	refresh := vault.NewRefresh(w.GetVaultReaders(), db.Stores.Vaults, notes, db.Source.RemovalLookup, db.Source.Sources, clock.System{})
	refresh.Cut = w.cutNotes()
	refresh.Ledger = db.Ledger()
	return refresh
}

// OpenVaultWith is how a vault is opened over the readers and the watcher given.
func (w readingParts) OpenVaultWith(
	db *testIndex, walkCountingReaders vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) *vault.VaultOpener {
	scan := vault.NewScan(walkCountingReaders, db.Stores.Vaults, db.Stores.Notes, db.NoteQueries.FileStats, db.Stores.Maintenance, db.Source.SourceBacklog)
	w.setNoteCut(&scan)
	scan.Walks = db.Walks()
	buffered := &indexnote.Buffered{NoteRepository: db.Stores.Notes}
	return vault.ConfigureVaultOpener(watcher, scan, buffered, w.makeRefresh(db, buffered))
}

// ReadWholeVault reads the notes, cuts the file texts and makes the vectors of a vault.
func (w readingParts) ReadWholeVault(
	ctx context.Context, db *testIndex, embedder searchsourceport.Source, v domain.Vault, rebuild bool,
) (indexing.ReadWholeVault, error) {
	fit := vectors.NewFitIndex(db.VectorIndex(), nil)
	fit.ShouldRebuildIndex = rebuild
	if err := fit.Execute(ctx, embedder); err != nil {
		return indexing.ReadWholeVault{}, err
	}
	if _, err := w.GetVaultReaders().Open(v); err != nil {
		return indexing.ReadWholeVault{}, err
	}
	refresh := w.GetRefresh(db)
	refresh.Force = true
	readNote := func(ctx context.Context, v domain.Vault, path string) error {
		res, err := refresh.Execute(ctx, v, []string{path})
		if err != nil {
			return err
		}
		if slices.Contains(res.Unreadable, path) {
			return fmt.Errorf("%s: the note could not be read", path)
		}
		return nil
	}
	texts := filetext.NewHandlingIndex(filetext.NewIndex(w.GetVaultReaders(), db.Source.Sources, db.Source.SourceRecords, db.Source.SourceBacklog, db.Source.RemovalLookup, w.texts), nil, readNote)
	cutter := w.cutNotes()
	texts.Sizes = cutter.Sizes
	texts.Legibility = cutter.Legibility
	texts.ShouldRebuildIndex = rebuild
	store, err := derivedfile.OpenDerived(v.Path, vaultconfig.Options{}, derivedCoreFolder)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	embed := vectors.NewEmbed(w.GetVaultReaders(), db.Chunks.VectorQueries, db.Chunks.Vectors)
	embed.Claim = store.NewIndexClaim(derivedCoreFolder + "/indexing")
	embed.Texts = w.texts
	embed.Source = embedder
	return indexing.NewReadWholeVault(w.scan(db, rebuild), texts, embed), nil
}

func openThemes(registryPath string, interfaceScale, textScale float64, say func(string)) (*theme.Service, error) {
	folder := filepath.Dir(registryPath)
	hub := vaultwatch.NewHub()
	files, err := themefiles.OpenAt(filepath.Join(folder, "themes"), hub.GetShared())
	sources := themeusecase.Sources{Presets: files, Own: files, Plugins: nil}
	getPath := func() (string, error) { return filepath.Join(folder, "numen.json"), nil }

	declarations := func(owner string) ([]manifest.Setting, bool) {
		if owner == settingsdomain.AppOwner {
			return settingsadapter.GetAppDeclaration(), true
		}
		return nil, false
	}
	registries := func(name string) ([]string, bool) {
		if name != settingsadapter.RegistryThemes {
			return nil, false
		}
		var names []string
		for _, one := range themeusecase.NewListThemes(sources).Execute() {
			names = append(names, one.Name)
		}
		return names, true
	}
	store := settingsadapter.NewStore(settingsadapter.NewFiles(), getPath, declarations, registries)
	list := settingsusecase.NewListSettingValues(
		settingsadapter.JSONCodec{}, store, settingsadapter.Defaults{}, settingsadapter.AppDeclaration{}, nil,
	)
	feed := settingevents.Feed{
		Events:  settingevents.New(),
		GetPath: getPath,
		Read:    list.Execute,
		Initial: func() settingsdomain.Values {
			values, err := list.ListDefaults()
			if err != nil {
				return settingsdomain.Values{}
			}
			return values
		},
		Report: func(error) {},
		Hub:    hub.GetShared(),
	}
	var noPlugins settingsport.PluginSettings
	check := settingsusecase.NewCheckSettings(
		settingsadapter.JSONCodec{}, settingsadapter.NewValidator(declarations, registries), noPlugins,
	)
	write := settingsusecase.NewWriteSettings(check, store, feed)

	appearance := settingsadapter.NewAppearanceStore(
		settingsusecase.NewReadAppearance(store).Execute,
		settingsusecase.NewWriteAppearance(settingsadapter.JSONCodec{}, write).Execute,
		say,
	)
	scales := themeusecase.NewScales(interfaceScale, textScale)
	interfaceBounds, textBounds := settingsadapter.GetInterfaceScaleBounds(), settingsadapter.GetTextScaleBounds()
	return theme.NewService(
		themeusecase.NewGetThemes(sources, appearance, appearance, scales,
			themeusecase.Bounds{Min: interfaceBounds.Min, Max: interfaceBounds.Max},
			themeusecase.Bounds{Min: textBounds.Min, Max: textBounds.Max}),
		themeusecase.NewReadTheme(sources),
		themeusecase.NewWriteAppearance(sources, appearance, appearance, scales),
		themeusecase.NewWatchThemes(sources, files, 0),
	), err
}
