package editor

import (
	"context"
	"fmt"
	"math"
	"slices"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"

	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	indexnote "github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/version"
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
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
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
func (w readingParts) OpenIndex(ctx context.Context) (*vault.Index, error) {
	db, err := index.Open(ctx, w.indexPath)
	if err != nil {
		return nil, err
	}
	queries := db.NoteQueries()
	return vault.NewIndex(vault.IndexPorts{
		Closer: db,
		File:   w.indexPath,
		Stores: vault.IndexStores{
			Vaults: db.Vaults(), Notes: db.Notes(), OpenTabs: db.OpenTabs(), Maintenance: db.Maintenance(),
		},
		NoteQueries: vault.IndexNoteQueries{
			FileStats: queries, NameMatches: queries, Headings: queries, Summaries: queries, NoteTypes: queries,
			TypedPaths: queries, VaultHolds: queries, NoteRefs: queries, OpeningNote: queries, NamedPaths: queries,
			Links: queries, Problems: db.NoteProblems(),
		},
		Chunks: vault.IndexChunks{
			Passages: db.ChunkSearch(), Nearest: db.ChunkSearch(), Progress: db.ChunkVectors(), Stats: db.ChunkVectors(),
			Index:   db,
			Vectors: db.Sources(), VectorQueries: db.Sources(),
		},
		Source: vault.IndexSources{
			Sources: db.Sources(), SourcesUnder: db.Sources(), RemovalLookup: db.Sources(),
			SourceRecords: db.Sources(), SourceBacklog: db.Sources(),
		},
		Search: vault.IndexSearch{
			SearchFloors: db.SearchFloors(), SearchSamples: db.SearchSamples(),
		},
		TextVersions: version.NewIndexVersions(db.TextVersions()),
	}), nil
}

// GetVaultReaders opens vaults for reading.
func (w readingParts) GetVaultReaders() vaultport.VaultReaders { return vaultreader.Readers{} }

// GetVaultWatcher monitors vaults for external filesystem changes.
func (w readingParts) GetVaultWatcher() vaultport.VaultWatcher {
	return vaultwatch.Watcher{Hub: w.watches.GetShared()}
}

func (w readingParts) cutNotes() chunking.CutNotes {
	return chunking.NewCutNotes(w.chunks.Get())
}

func (w readingParts) setNoteCut(scan *vault.Scan, db *vault.Index) {
	scan.Cut = w.cutNotes()
	scan.ShouldRecut = w.chunks.Get() > 0
	scan.GetOtherRecipePaths = func(ctx context.Context, v domain.VaultID, recipe string) ([]string, error) {
		recipes := []string{recipe, filetext.Index{Sizes: scan.Cut.Sizes}.Recipe()}
		return db.Source.SourceBacklog.GetSourcesWithOtherRecipe(ctx, v, domain.KindNote, recipes, math.MaxInt32)
	}
}

func (w readingParts) scan(db *vault.Index, rebuild bool) vault.Scan {
	scan := vault.NewScan(w.GetVaultReaders(), db.Stores.Vaults, db.Stores.Notes, db.NoteQueries.FileStats, db.Stores.Maintenance)
	w.setNoteCut(&scan, db)
	scan.Walks = db.Walks()
	scan.ShouldRebuildIndex = rebuild
	return scan
}

// GetRefresh brings named notes up to date in the index, over the notes of the cache.
func (w readingParts) GetRefresh(db *vault.Index) vault.Refresh {
	return w.makeRefresh(db, db.Stores.Notes)
}

func (w readingParts) makeRefresh(db *vault.Index, notes noteport.NoteRepository) vault.Refresh {
	refresh := vault.NewRefresh(w.GetVaultReaders(), db.Stores.Vaults, notes, db.Source.RemovalLookup, db.Source.Sources, clock.System{})
	refresh.Cut = w.cutNotes()
	refresh.Ledger = db.Ledger()
	return refresh
}

// OpenVaultWith is how a vault is opened over the readers and the watcher given.
func (w readingParts) OpenVaultWith(
	db *vault.Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) *vault.VaultOpener {
	scan := vault.NewScan(walking, db.Stores.Vaults, db.Stores.Notes, db.NoteQueries.FileStats, db.Stores.Maintenance)
	w.setNoteCut(&scan, db)
	scan.Walks = db.Walks()
	buffered := &indexnote.Buffered{NoteRepository: db.Stores.Notes}
	return vault.ConfigureVaultOpener(watcher, scan, buffered, w.makeRefresh(db, buffered))
}

// ReadWholeVault reads the notes, cuts the file texts and makes the vectors of a vault.
func (w readingParts) ReadWholeVault(
	ctx context.Context, db *vault.Index, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
) (indexing.ReadWholeVault, error) {
	fit := vectors.NewFitIndex(db.VectorIndex())
	fit.ShouldRebuildIndex = rebuild
	if err := fit.Execute(ctx, embedder); err != nil {
		return indexing.ReadWholeVault{}, err
	}
	if _, err := w.GetVaultReaders().Open(v); err != nil {
		return indexing.ReadWholeVault{}, err
	}
	texts := filetext.NewIndex(w.GetVaultReaders(), db.Source.Sources, db.Source.SourceRecords, db.Source.SourceBacklog, db.Source.RemovalLookup, w.texts)
	cutter := w.cutNotes()
	texts.Sizes = cutter.Sizes
	texts.Legibility = cutter.Legibility
	texts.ShouldRebuildIndex = rebuild
	refresh := w.GetRefresh(db)
	refresh.Force = true
	texts.ReadNote = func(ctx context.Context, v domain.Vault, path string) error {
		res, err := refresh.Execute(ctx, v, []string{path})
		if err != nil {
			return err
		}
		if slices.Contains(res.Unreadable, path) {
			return fmt.Errorf("%s: the note could not be read", path)
		}
		return nil
	}
	store, err := derivedfile.OpenDerived(v.Path, vaultconfig.Options{}, derivedCoreFolder)
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	embed := vectors.NewEmbed(w.GetVaultReaders(), db.Chunks.VectorQueries, db.Chunks.Vectors)
	embed.Claim = store.NewIndexClaim(derivedCoreFolder + "/indexing")
	embed.Texts = w.texts
	embed.Embedder = embedder
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
	store := settingsadapter.NewStore(settingsadapter.NewFiles(), getPath, declarations, registries, nil)
	list := settingsusecase.NewListSettingValues(
		settingsadapter.JSONCodec{}, store, settingsadapter.Defaults{}, settingsadapter.AppDeclaration{}, nil, nil,
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
