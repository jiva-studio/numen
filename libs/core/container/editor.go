package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/usecase/source"
	"github.com/jiva-studio/numen/libs/core/usecase/transcription"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/downloadport"
	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/indexport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/recognitionport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// EditorAssembly is this installation as the window a person writes in asks it:
// the vault list, the cache baseline as the ports the window reaches it through,
// and the scenarios and passes a vault is opened with.
//
// It is what the window is handed. The window declares what it asks for and is
// given it, so nothing of the window's is named here.
type EditorAssembly struct {
	cfg      Config
	registry vaultport.VaultRegistry
	db       *Index
}

// GetEditorAssembly opens the cache and binds everything that window relies on
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
func NewEditorAssembly(cfg Config, registry vaultport.VaultRegistry, db *Index) EditorAssembly {
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

func (e EditorAssembly) EmitEvent(event string) {
	if e.cfg.PluginHost != nil {
		var emitter pluginport.PluginEventEmitter = e.cfg.PluginHost
		emitter.EmitEvent(event)
	}
}

func (e EditorAssembly) GetRegistry() vaultport.VaultRegistry { return e.registry }

func (e EditorAssembly) GetVaultReaders() vaultport.VaultReaders   { return e.cfg.VaultReaders() }
func (e EditorAssembly) GetVaultWriters() vaultport.VaultWriters   { return e.cfg.VaultWriters() }
func (e EditorAssembly) GetDerivedStores() vaultport.DerivedStores { return e.cfg.GetDerivedStores() }

func (e EditorAssembly) GetBooks() sourceport.Book                  { return e.cfg.Books() }
func (e EditorAssembly) GetTextExtractor() sourceport.TextExtractor { return e.cfg.TextExtractor() }
func (e EditorAssembly) GetPageRenderer() sourceport.PageRenderer   { return e.cfg.PageRenderer() }
func (e EditorAssembly) GetVaultIdentity() vaultport.VaultIdentity  { return e.cfg.VaultIdentity() }
func (e EditorAssembly) GetClock() machineport.Clock                { return e.cfg.Clock() }

func (e EditorAssembly) GetNoteQueries() noteport.NoteQueries              { return e.db.Queries() }
func (e EditorAssembly) GetLinkQueries() noteport.LinkQueries              { return e.db.Links() }
func (e EditorAssembly) GetProblemQueries() recognitionport.ProblemQueries { return e.db.Problems() }
func (e EditorAssembly) GetIndexProgress() indexport.IndexProgress         { return e.db.Progress() }

func (e EditorAssembly) GetSourceRepository() sourceport.SourceRepository { return e.db.Sources() }
func (e EditorAssembly) GetSourceQueries() sourceport.SourceQueries       { return e.db.SourcesKnown() }
func (e EditorAssembly) GetPassageQueries() sourceport.PassageQueries     { return e.db.Passages() }

func (e EditorAssembly) CloseIndex() error { return e.db.Close() }

func (e EditorAssembly) OpenGetSettings() *settings.Get {
	target := e.cfg.NestedHeadingLimitBounds()
	return settings.NewGet(
		e.cfg.ReadSettings(), e.cfg.Models(), settings.Bounds{Least: target.Least, Most: target.Most},
		e.cfg.Plugins(),
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

// OpenRetryPlugin is nil where the retry cannot be made, which the window answers as served by no build.
func (e EditorAssembly) OpenRetryPlugin() *settings.RetryPlugin {
	retry, err := settings.NewRetryPlugin(e.cfg.GetPluginRetrier())
	if err != nil {
		return nil
	}
	return retry
}

func (e EditorAssembly) ReportPluginFailures(tasks *run.Tasks) {
	e.cfg.ReportPluginFailures(tasks)
	e.cfg.ReportDependencies(tasks)
}

func (e EditorAssembly) SetPluginWork(tasks *run.Tasks, models *run.Lock) {
	e.cfg.SetPluginWork(tasks, models)
}

func (e EditorAssembly) IsRebuildingIndex() bool     { return e.cfg.ShouldRebuildIndex }
func (e EditorAssembly) IsTranscribingUnasked() bool { return e.cfg.ShouldTranscribe }

func (e EditorAssembly) OpenEmbedders(
	ctx context.Context, tasks *run.Tasks,
) (indexing, asking embeddingport.Embedder, closer func() error, why error) {
	return e.cfg.Embedders(ctx, tasks)
}

func (e EditorAssembly) NewSearch(
	asking embeddingport.Embedder, handleError func(error),
) search.Search {
	return e.cfg.NewSearch(e.db, asking, handleError)
}

func (e EditorAssembly) OpenNotes(index note.Levels) note.Scenarios {
	return e.cfg.Notes(e.db.Queries(), e.db.Links(), e.db.Sources(), e.db.SourcesKnown(), index)
}

func (e EditorAssembly) OpenVaults(moves note.Move) vault.Scenarios {
	return e.cfg.Vaults(e.registry, e.db, moves)
}

func (e EditorAssembly) OpenDownloader(ctx context.Context) downloadport.Downloader {
	return e.cfg.Downloader(ctx)
}

func (e EditorAssembly) OpenImportURL(
	ctx context.Context, downloader downloadport.Downloader,
) source.ImportURL {
	return e.cfg.ImportURL(ctx, e.db, downloader)
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
	ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
) (vault.ReadWholeVault, error) {
	request := e.cfg
	request.ShouldRebuildIndex = rebuild
	return request.ReadWholeVault(ctx, e.db, embedder, v)
}

// StartVault opens the vault the way both windows open one, and answers with
// the walk, the watch, the levelling and why the vault is not being followed.
func (e EditorAssembly) StartVault(
	ctx context.Context, v domain.Vault, rebuild bool,
	notified func(paths, assets []string, reload bool),
	handleError windowport.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vault.Refresh,
	unwatched error,
) {
	request := e.cfg
	request.ShouldRebuildIndex = rebuild

	opening := request.VaultOpener(e.db)
	opening.ShouldRebuild = rebuild
	opening.ErrorHandler = handleError
	if notified != nil {
		opening.Told = func(m VaultChanges) { notified(m.Paths, m.Assets, m.ShouldReload) }
	}

	open := opening.Begin(ctx, v)
	return open.Read, open.Run, opening.GetRefresh(), open.GetUnwatchedReason()
}

// GetPluginServices is where the window forwards the calls it receives for the services plugins
// serve.
func (e EditorAssembly) GetPluginServices() PluginServices { return e.cfg.GetPluginServices() }

// GetAgent returns the agent provider for running tasks.
func (e EditorAssembly) GetAgent() agentport.Agent { return e.cfg.AgentProvider() }
