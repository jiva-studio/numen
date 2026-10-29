package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/assembly"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/indexport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/problemport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// EditorAssembly is this installation as the window a person writes in asks it.
type EditorAssembly = assembly.Runtime[PluginVaultHooks, pluginport.PluginVault]

// UnknownVaultError is a vault named at start that the registry does not hold.
type UnknownVaultError = vault.UnknownError

// GetEditorAssembly opens the cache and binds everything that window relies on
// to this installation's adapters.
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
	provider := cfg.AgentProvider()
	return EditorAssembly{
		PluginDirFn:        cfg.GetPluginDir,
		EmitEventFn:        cfg.emitPluginEvent,
		RegistryVal:        registry,
		VaultReadersFn:     cfg.VaultReaders,
		VaultWritersFn:     cfg.VaultWriters,
		VaultIdentityFn:    cfg.VaultIdentity,
		OpenTabsStoreFn:    func() windowport.OpenTabsStore { return db.OpenTabs() },
		ClockFn:            cfg.Clock,
		NoteQueriesFn:      func() noteport.NoteQueries { return db.Queries() },
		LinkQueriesFn:      func() noteport.LinkQueries { return db.Links() },
		ProblemQueriesFn:   func() problemport.ProblemQueries { return db.Problems() },
		IndexProgressFn:    func() indexport.IndexProgress { return db.Progress() },
		SourceRepositoryFn: func() sourceport.SourceRepository { return db.Sources() },
		SourceQueriesFn:    func() sourceport.SourceQueries { return db.SourcesKnown() },
		PassageQueriesFn:   func() sourceport.PassageQueries { return db.Passages() },
		CloseIndexFn:       func() error { return db.Close() },
		OpenGetSettingsFn: func() *settings.Get {
			return settings.NewGet(cfg.ReadSettings(), cfg.modelsForProvider(provider), cfg.Plugins())
		},
		OpenWriteSettingsFn:      func() *settings.Write { return settings.NewWrite(cfg.GetSettingsWriter()) },
		OpenWritePluginSecretFn:  func() *settings.WriteSecret { return settings.NewWriteSecret(cfg.GetPluginSecretWriter()) },
		OpenDeletePluginSecretFn: func() *settings.DeleteSecret { return settings.NewDeleteSecret(cfg.GetPluginSecretDeleter()) },
		OpenRetryPluginFn:        cfg.openRetryPlugin,
		ReportPluginFailuresFn:   func(tasks *run.Tasks) { cfg.ReportPluginFailures(tasks); cfg.ReportDependencies(tasks) },
		SetPluginWorkFn:          cfg.SetPluginWork,
		IsRebuildingIndexVal:     cfg.ShouldRebuildIndex,
		OpenEmbeddersFn:          cfg.Embedders,
		NewSearchFn: func(asking embeddingport.Embedder, handleError func(error)) search.Search {
			return cfg.NewSearch(db, asking, handleError)
		},
		OpenNotesFn: func(index note.Levels) note.Scenarios {
			return cfg.Notes(db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), index)
		},
		OpenVaultsFn: func(moves note.Move) vault.Scenarios {
			return cfg.Vaults(registry, db, moves)
		},
		GetRefreshFn: func() vault.Refresh {
			return makeRefresh(cfg, db, db.Notes())
		},
		ReadWholeVaultFn: func(ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool) (vault.ReadWholeVault, error) {
			request := cfg
			request.ShouldRebuildIndex = rebuild
			return request.ReadWholeVault(ctx, db, embedder, v)
		},
		StartVaultFn:     cfg.startVault(db),
		PluginServicesFn: cfg.GetPluginServices,
		PluginCommandsFn: cfg.GetPluginCommands,
		AgentVal:         cfg.serveTools(provider),
		SetTextVersionListenerFn: func(listener textport.VersionListener) {
			if cfg.TextVersions != nil {
				cfg.TextVersions.SetListener(listener)
			}
		},
		OpenFileMenuFn: cfg.openFileMenu,
		OpenProxyFn: func() *file.OpenProxy {
			return file.NewOpenProxy(cfg.VaultReaders(), cfg.Copies())
		},
		GetMediaTypeFn: func() *file.GetMediaType {
			return file.NewGetMediaType(sharedmimeinfo.Open())
		},
		OpenChoosePluginPathFn: func(dialog machineport.PathDialog) *settings.ChoosePluginPath {
			return settings.NewChoosePluginPath(dialog, cfg.Plugins(), cfg.GetSettingsWriter())
		},
		OpenFileTextFn:   cfg.openFileText,
		OpenPluginDataFn: cfg.openPluginData,
		OpenPluginVaultFn: func(current func() (domain.Vault, bool), hooks PluginVaultHooks) (pluginport.PluginVault, error) {
			return cfg.OpenPluginVault(registry, db, current, hooks)
		},
	}
}

func (c Config) emitPluginEvent(event string) {
	if c.PluginHost != nil {
		var emitter pluginport.PluginEventEmitter = c.PluginHost
		emitter.EmitEvent(event)
	}
}

func (c Config) openRetryPlugin() *settings.RetryPlugin {
	retry, err := settings.NewRetryPlugin(c.GetPluginRetrier())
	if err != nil {
		return nil
	}
	return retry
}

func (c Config) startVault(db *Index) func(
	ctx context.Context, v domain.Vault, rebuild bool,
	notified func(paths, assets []string, reload bool),
	handleError windowport.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vault.Refresh,
	unwatched error,
) {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool,
		notified func(paths, assets []string, reload bool),
		handleError windowport.ErrorHandler,
	) (
		read func(ctx context.Context, during func()) (notes int, err error),
		run func(ctx context.Context),
		refresh vault.Refresh,
		unwatched error,
	) {
		request := c
		request.ShouldRebuildIndex = rebuild

		opening := request.VaultOpener(db)
		opening.ShouldRebuild = rebuild
		opening.ErrorHandler = handleError
		if notified != nil {
			opening.Told = func(m VaultChanges) { notified(m.Paths, m.Assets, m.ShouldReload) }
		}

		open := opening.Begin(ctx, v)
		return open.Read, open.Run, opening.GetRefresh(), open.GetUnwatchedReason()
	}
}

func (c Config) openFileMenu() *textversion.FileMenu {
	if c.TextVersions == nil {
		return nil
	}
	var commands pluginport.PluginCommands = commandbridge.NoCommands{}
	var types contenttypeport.ContentTypeTable = sharedmimeinfo.Open()
	if c.PluginHost != nil && !c.SafeMode {
		commands, types = commandbridge.Open(c.PluginHost), c.PluginHost
	}
	menu, err := textversion.NewFileMenu(c.TextVersions, commands, types, c.VaultReaders())
	if err != nil {
		c.handleError(err)
		return nil
	}
	return menu
}

func (c Config) openFileText() *textversion.FileText {
	if c.TextVersions == nil {
		return nil
	}
	text, err := textversion.NewFileText(c.TextVersions, c.VaultReaders())
	if err != nil {
		c.handleError(err)
		return nil
	}
	return text
}
