package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/assembly"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/indexport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/problemport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// EditorAssembly is this installation as the editor window requests it.
type EditorAssembly = assembly.Runtime[PluginVaultHooks, pluginport.PluginVault]

// UnknownVaultError is a vault named at start that the registry does not hold.
type UnknownVaultError = registry.UnknownError

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
func NewEditorAssembly(cfg Config, ports registry.Ports, db *Index) EditorAssembly {
	provider := cfg.AgentProvider()
	return EditorAssembly{
		PluginDirFn:    cfg.GetPluginDir,
		EmitEventFn:    cfg.plugins().EmitPluginEvent,
		VaultReadersFn: cfg.VaultReaders,
		OpenGetVaultToShowFn: func() registry.GetVaultToShow {
			return registry.NewGetVaultToShow(ports.Lookup, ports.Vaults, ports.Last)
		},
		OpenCreateFolderFn:   func() *file.CreateFolder { return file.NewCreateFolder(cfg.vault().FolderWriters()) },
		VaultIdentityFn:      cfg.VaultIdentity,
		OpenTabsStoreFn:      func() windowport.OpenTabsStore { return db.OpenTabs() },
		ClockFn:              cfg.Clock,
		OpenGetOpeningNoteFn: func() *query.GetOpeningNote { return query.NewGetOpeningNote(db.OpeningNote()) },
		OpenGetTypesFn:       func() *query.GetTypes { return query.NewGetTypes(db.NoteTypes()) },
		OpenGetHeadingsFn:    func() *query.GetHeadings { return query.NewGetHeadings(db.Headings()) },
		OpenResolveLinksFn:   func() *query.ResolveLinks { return query.NewResolveLinks(db.Links()) },
		OpenGetTitlesFn:      func() query.GetTitles { return query.NewGetTitles(db.NoteRefs()) },
		OpenGetNameMatchesFn: func() *search.GetNameMatches { return search.NewGetNameMatches(db.NameMatches()) },
		OpenGetFolderEntriesFn: func(readers vaultport.VaultReaders) *file.GetFolderEntries {
			return file.NewGetFolderEntries(readers, db.NoteTypes())
		},
		OpenGetFileKindsFn: func(readers vaultport.VaultReaders) *file.GetFileKinds {
			return file.NewGetFileKinds(readers, db.NoteTypes())
		},
		OpenFocusNoteFn:    func(view windowport.Window) *query.FocusNote { return cfg.FocusNote(db.NoteRefs(), view) },
		ProblemQueriesFn:   func() problemport.ProblemQueries { return db.Problems() },
		IndexProgressFn:    func() indexport.IndexProgress { return db.Progress() },
		SourceRepositoryFn: func() sourceport.SourceRepository { return db.Sources() },
		SourceQueriesFn:    func() sourceport.SourceQueries { return db.SourcesKnown() },
		PassageQueriesFn:   func() sourceport.PassageQueries { return db.Passages() },
		CloseIndexFn:       func() error { return db.Close() },
		OpenGetSettingsFn: func() *settings.Get {
			return settings.NewGet(cfg.ReadSettings(), cfg.modelsForProvider(provider), cfg.Plugins(), cfg.settings().ListAppSettings)
		},
		OpenWriteSettingsFn:    cfg.settings().OpenWriteSettings,
		OpenWatchSettingsFn:    cfg.settings().OpenWatchSettings,
		OpenSecretsFn:          cfg.OpenSecrets,
		OpenRetryPluginFn:      func() *settings.RetryPlugin { return cfg.settings().OpenRetryPlugin(cfg.GetPluginRetrier()) },
		ReportPluginFailuresFn: func(tasks *run.Tasks) { cfg.ReportPluginFailures(tasks); cfg.ReportDependencies(tasks) },
		SetPluginWorkFn:        cfg.SetPluginWork,
		IsRebuildingIndexVal:   cfg.ShouldRebuildIndex,
		OpenEmbeddersFn:        cfg.Embedders,
		NewSearchFn: func(asking embeddingport.Embedder, handleError func(error)) search.Search {
			return cfg.NewSearch(db, asking, handleError)
		},
		SetPluginSearchFn: func(embedder embeddingport.Embedder, handleError func(error)) {
			cfg.SetPluginSearch(db, embedder, handleError)
		},
		OpenNotesFn: func(index note.IndexUpdater) note.Scenarios {
			return cfg.Notes(db, index)
		},
		OpenVaultsFn: func(moves note.Move) vault.Scenarios {
			return cfg.Vaults(ports, db, moves)
		},
		GetRefreshFn: func() vault.Refresh {
			return makeRefresh(cfg, db, db.Notes())
		},
		ReadWholeVaultFn: func(ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool) (vault.ReadWholeVault, error) {
			request := cfg
			request.ShouldRebuildIndex = rebuild
			return request.ReadWholeVault(ctx, db, embedder, v)
		},
		StartVaultFn:             cfg.startVault(db),
		PluginServicesFn:         cfg.GetPluginServices,
		PluginCommandsFn:         cfg.GetPluginCommands,
		OpenChooseTabKindFn:      cfg.OpenChooseTabKind,
		PluginCatalogFn:          cfg.getEditorCatalog,
		AgentVal:                 cfg.plugins().GetToolProvider(),
		SetTextVersionListenerFn: cfg.vault().SetTextVersionListener,
		ReportTextFilesChangedFn: cfg.vault().ReportTextFilesChanged,
		OpenVersionItemsFn:       cfg.plugins().OpenVersionItems,
		WatchPluginChangesFn:     cfg.WatchPluginChanges,
		OpenTriggersFn:           cfg.plugins().OpenTriggers,
		OpenPluginFileFn:         cfg.vault().OpenPluginFile,
		GetMediaTypeFn:           cfg.vault().GetMediaType,
		OpenChoosePluginPathFn: func(dialog machineport.PathDialog) *settings.ChoosePluginPath {
			return cfg.settings().OpenChoosePluginPath(dialog)
		},
		OpenFileTextFn:   cfg.vault().OpenFileText,
		OpenPluginDataFn: cfg.vault().OpenPluginData,
		OpenPluginVaultFn: func(current func() (domain.Vault, bool), hooks PluginVaultHooks) (pluginport.PluginVault, error) {
			return cfg.OpenPluginVault(ports, db, current, hooks)
		},
	}
}

func (c Config) startVault(db *Index) func(
	ctx context.Context, v domain.Vault, rebuild bool,
	notified func(paths, assets []string, reload bool),
	handleError window.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vault.Refresh,
	unwatched error,
) {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool,
		notified func(paths, assets []string, reload bool),
		handleError window.ErrorHandler,
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
			opening.Told = func(m vault.VaultChanges) { notified(m.Paths, m.Assets, m.ShouldReload) }
		}

		open := opening.Begin(ctx, v)
		return open.Read, open.Run, opening.GetRefresh(), open.GetUnwatchedReason()
	}
}
