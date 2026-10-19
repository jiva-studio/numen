package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// EditorAssembly is this installation as the window a person writes in asks it:
// the vault list, the cache baseline as the ports the window reaches it through,
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

func (e EditorAssembly) GetPluginDir(id string) (string, bool) {
	if e.cfg.PluginHost == nil {
		return "", false
	}
	p, ok := e.cfg.PluginHost.GetPlugin(id)
	if !ok {
		return "", false
	}
	return p.Dir, true
}

func (e EditorAssembly) OpenBookClient(ctx context.Context) (any, func(), error) {
	host, err := e.cfg.resolvePlugin(plugin.CapabilityBook)
	if err != nil {
		return nil, nil, err
	}
	bridge, err := plugin.OpenBookBridge(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	return bridge, func() { _ = bridge.Close() }, nil
}

func (e EditorAssembly) EmitEvent(event string) {
	if e.cfg.PluginHost != nil {
		var emitter port.PluginEventEmitter = e.cfg.PluginHost
		emitter.EmitEvent(event)
	}
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

func (e EditorAssembly) OpenGetSettings(day flashcards.Day) *settings.Get {
	held := e.cfg.NestedHeadingLimitBounds()
	return settings.NewGet(
		e.cfg.ReadSettings(), e.cfg.Models(), settings.Bounds{Least: held.Least, Most: held.Most},
		e.cfg.GetLatestDayStart(), day, e.cfg.Clock(), e.cfg.Plugins(),
	)
}

func (e EditorAssembly) OpenWriteSettings() *settings.Write {
	return settings.NewWrite(e.cfg.GetSettingsWriter())
}

func (e EditorAssembly) OpenGetSettingsFile() *settings.GetFile {
	return settings.NewGetFile(e.cfg.ReadSettingsFile())
}

func (e EditorAssembly) OpenWriteSettingsFile() *settings.WriteFile {
	return settings.NewWriteFile(e.cfg.WriteConfiguredFile())
}

func (e EditorAssembly) OpenWritePluginSecret() *settings.WriteSecret {
	return settings.NewWriteSecret(e.cfg.GetPluginSecretWriter())
}

func (e EditorAssembly) OpenDeletePluginSecret() *settings.DeleteSecret {
	return settings.NewDeleteSecret(e.cfg.GetPluginSecretDeleter())
}

func (e EditorAssembly) IsRebuildingIndex() bool     { return e.cfg.ShouldRebuildIndex }
func (e EditorAssembly) IsTranscribingUnasked() bool { return e.cfg.ShouldTranscribe }

func (e EditorAssembly) OpenEmbedders(
	ctx context.Context, tasks *run.Tasks,
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

func (e EditorAssembly) OpenSourceTexts() source.Texts {
	return e.cfg.SourceTexts(e.db.SourcesKnown())
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
	ctx context.Context, tasks *run.Tasks, models *run.Lock,
) *recognition.Worker {
	return e.cfg.OpenRecognitionWorker(ctx, e.db.Sources(), tasks, models)
}

func (e EditorAssembly) OpenTranscriptionWorker(
	ctx context.Context, tasks *run.Tasks, models *run.Lock,
) *transcription.Worker {
	return e.cfg.OpenTranscriptionWorker(ctx, e.db.Sources(), tasks, models)
}

func (e EditorAssembly) GetRefresh() vault.Refresh {
	return makeRefresh(e.cfg, e.db, e.db.Notes())
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
