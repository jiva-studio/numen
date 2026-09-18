package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// EditorAssembly is this installation as the window a person writes in asks it:
// the vault list, the cache seen as the ports the window reaches it through,
// and the scenarios and passes a vault is opened with.
//
// It is what the window is handed. The window declares what it asks for and is
// given it, so nothing of the window's is named here.
type EditorAssembly struct {
	cfg      Config
	registry port.VaultRegistry
	db       *Index
}

// GetEditorAssembly opens the cache and binds everything that window stands on
// to this installation's adapters. The window closes the cache when it closes,
// and a window that could not be opened closes it too.
func (c Config) GetEditorAssembly(ctx context.Context) (EditorAssembly, error) {
	registry, err := c.Registry()
	if err != nil {
		return EditorAssembly{}, err
	}
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return EditorAssembly{}, err
	}
	return NewEditorAssembly(c, registry, db), nil
}

// NewEditorAssembly is that assembly over a list of vaults and a cache already
// open, for a caller that opened them itself.
func NewEditorAssembly(cfg Config, registry port.VaultRegistry, db *Index) EditorAssembly {
	return EditorAssembly{cfg: cfg, registry: registry, db: db}
}

func (e EditorAssembly) GetRegistry() port.VaultRegistry { return e.registry }

func (e EditorAssembly) GetVaultReaders() port.VaultReaders   { return e.cfg.VaultReaders() }
func (e EditorAssembly) GetVaultWriters() port.VaultWriters   { return e.cfg.VaultWriters() }
func (e EditorAssembly) GetDerivedStores() port.DerivedStores { return e.cfg.GetDerivedStores() }

func (e EditorAssembly) GetTextExtractor() port.TextExtractor { return e.cfg.TextExtractor() }
func (e EditorAssembly) GetPageRenderer() port.PageRenderer   { return e.cfg.PageRenderer() }
func (e EditorAssembly) GetVaultIdentity() port.VaultIdentity { return e.cfg.VaultIdentity() }
func (e EditorAssembly) GetClock() port.Clock                 { return e.cfg.Clock() }

func (e EditorAssembly) GetNoteQueries() port.NoteQueries       { return e.db.Queries() }
func (e EditorAssembly) GetLinkQueries() port.LinkQueries       { return e.db.Links() }
func (e EditorAssembly) GetProblemQueries() port.ProblemQueries { return e.db.Problems() }
func (e EditorAssembly) GetIndexProgress() port.IndexProgress   { return e.db.Progress() }

func (e EditorAssembly) GetSourceRepository() port.SourceRepository { return e.db.Sources() }
func (e EditorAssembly) GetSourceQueries() port.SourceQueries       { return e.db.SourcesKnown() }
func (e EditorAssembly) GetPassageQueries() port.PassageQueries     { return e.db.Passages() }

func (e EditorAssembly) CloseIndex() error { return e.db.Close() }

func (e EditorAssembly) ReadSettings() (string, string, error)     { return e.cfg.ReadSettings()() }
func (e EditorAssembly) ReadSettingsFile() (string, string, error) { return e.cfg.ReadSettingsFile()() }
func (e EditorAssembly) GetModels() []port.Model                   { return e.cfg.Models()() }

func (e EditorAssembly) WriteSettingsFile(written string, seen *string) error {
	return e.cfg.WriteConfiguredFile()(written, seen)
}

func (e EditorAssembly) TurnSettings(written []port.Setting) error {
	return e.cfg.TurnsSetting()(written)
}

func (e EditorAssembly) GetPartsUnderANodeBounds() (least, most float64) {
	held := e.cfg.PartsUnderANodeBounds()
	return held.Least, held.Most
}

func (e EditorAssembly) GetLatestDayStarts() string { return e.cfg.GetLatestDayStart() }

func (e EditorAssembly) IsRebuildingIndex() bool     { return e.cfg.ShouldRebuildIndex }
func (e EditorAssembly) IsTranscribingUnasked() bool { return e.cfg.ShouldTranscribe }

func (e EditorAssembly) OpenEmbedders(
	ctx context.Context, tasks *task.Tasks,
) (indexing, asking port.Embedder, closer func() error, why error) {
	return e.cfg.Embedders(ctx, tasks)
}

func (e EditorAssembly) NewSearch(
	asking port.Embedder, handleError func(error),
) search.Search {
	return e.cfg.NewSearch(e.db, asking, handleError)
}

func (e EditorAssembly) OpenNotes(index note.Levels) note.Scenarios {
	return e.cfg.Notes(e.db.Queries(), e.db.Links(), e.db.Sources(), e.db.SourcesKnown(), index)
}

func (e EditorAssembly) OpenCards(index note.Levels) cards.Scenarios {
	return e.cfg.Cards(e.db.Queries(), e.db.Links(), index)
}

func (e EditorAssembly) OpenVaults(moves note.Move) vault.Scenarios {
	return e.cfg.Vaults(e.registry, e.db, moves)
}

func (e EditorAssembly) OpenFlashcards(index note.Levels) flashcards.Scenarios {
	return e.cfg.Flashcards(e.db.Queries(), e.db.Links(), e.db.Problems(), index)
}

func (e EditorAssembly) OpenDownloader(ctx context.Context) port.Downloader {
	return e.cfg.Downloader(ctx)
}

func (e EditorAssembly) OpenImportURL(
	ctx context.Context, by port.Downloader,
) source.ImportURL {
	return e.cfg.ImportURL(ctx, e.db, by)
}

func (e EditorAssembly) OpenRecognitionWorker(
	ctx context.Context, tasks *task.Tasks, models *source.Lock,
) *source.RecognitionWorker {
	return e.cfg.OpenRecognitionWorker(ctx, e.db.Sources(), tasks, models)
}

func (e EditorAssembly) OpenTranscriptionWorker(
	ctx context.Context, tasks *task.Tasks, models *source.Lock,
) *source.TranscriptionWorker {
	return e.cfg.OpenTranscriptionWorker(ctx, e.db.Sources(), tasks, models)
}

func (e EditorAssembly) GetRefresh() vault.Refresh {
	return makeRefresh(e.cfg, e.db,
		e.db.NotesCutAt(e.cfg.GetChunkSizes(), e.cfg.Legibility()))
}

// ReadWholeVault reads the vault the one way, with this launch's answer about
// reading every file again standing for the one pass that was asked for it.
func (e EditorAssembly) ReadWholeVault(
	ctx context.Context, embedder port.Embedder, v domain.Vault, rebuild bool,
) (vault.ReadWholeVault, error) {
	asked := e.cfg
	asked.ShouldRebuildIndex = rebuild
	return asked.ReadWholeVault(ctx, e.db, embedder, v)
}

// StartVault opens the vault the way both windows open one, and answers with
// the walk, the watch, the levelling and why the vault is not being followed.
func (e EditorAssembly) StartVault(
	ctx context.Context, v domain.Vault, rebuild bool,
	told func(paths, assets []string, reload bool),
	handleError port.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vault.Refresh,
	unwatched error,
) {
	asked := e.cfg
	asked.ShouldRebuildIndex = rebuild

	opening := asked.VaultOpener(e.db)
	opening.ShouldRebuild = rebuild
	opening.ErrorHandler = handleError
	if told != nil {
		opening.Told = func(m VaultChanges) { told(m.Paths, m.Assets, m.ShouldReload) }
	}

	open := opening.Begin(ctx, v)
	return open.Read, open.Run, opening.GetRefresh(), open.GetUnwatchedReason()
}
