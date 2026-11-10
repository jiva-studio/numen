package assembly

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	agentport "github.com/jiva-studio/numen/libs/core/kernel/usecase/completion/agentcontract"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	machineport "github.com/jiva-studio/numen/libs/core/kernel/usecase/file/machinecontract"
	sourceport "github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext/sourcecontract"
	indexport "github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing/indexcontract"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	problemport "github.com/jiva-studio/numen/libs/core/kernel/usecase/note/problemcontract"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	textport "github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion/textcontract"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/trigger"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	vaultport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/vaultcontract"
	windowport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/windowcontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors/embeddingcontract"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// StartVaultFunc is the signature for starting a vault session.
type StartVaultFunc func(
	ctx context.Context, v domain.Vault, rebuild bool,
	onSync func(paths, assets []string, reload bool),
	handleError window.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vaults.Refresh,
	unwatched error,
)

// Runtime bundles callbacks for an editor window assembly.
type Runtime[H, V any] struct {
	PluginDirFn              func(id string) (string, bool)
	EmitEventFn              func(event string)
	RegistryVal              vaultport.VaultRegistry
	VaultReadersFn           func() vaultport.VaultReaders
	VaultWritersFn           func() vaultport.VaultWriters
	VaultIdentityFn          func() vaultport.VaultIdentity
	OpenTabsStoreFn          func() windowport.OpenTabsStore
	ClockFn                  func() machineport.Clock
	OpenGetOpeningNoteFn     func() *query.GetOpeningNote
	OpenGetTypesFn           func() *query.GetTypes
	OpenGetHeadingsFn        func() *query.GetHeadings
	OpenResolveLinksFn       func() *query.ResolveLinks
	OpenGetTitlesFn          func() query.GetTitles
	OpenGetNameMatchesFn     func() *search.GetNameMatches
	OpenGetFolderEntriesFn   func(readers vaultport.VaultReaders) *file.GetFolderEntries
	OpenGetFileKindsFn       func(readers vaultport.VaultReaders) *file.GetFileKinds
	OpenFocusNoteFn          func(view windowport.Window) *query.FocusNote
	ProblemQueriesFn         func() problemport.ProblemQueries
	IndexProgressFn          func() indexport.IndexProgress
	SourceRepositoryFn       func() sourceport.SourceRepository
	SourceQueriesFn          func() sourceport.SourceQueries
	PassageQueriesFn         func() sourceport.PassageQueries
	CloseIndexFn             func() error
	OpenGetSettingsFn        func() *settings.Get
	OpenWriteSettingsFn      func() *settings.Write
	OpenWatchSettingsFn      func() *settings.Watch
	OpenSecretsFn            func() *secret.Secrets
	OpenRetryPluginFn        func() *settings.RetryPlugin
	ReportPluginFailuresFn   func(tasks *run.Tasks)
	SetPluginWorkFn          func(tasks *run.Tasks, models *run.Lock)
	IsRebuildingIndexVal     bool
	OpenEmbeddersFn          func(ctx context.Context, tasks *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error)
	NewSearchFn              func(asking embeddingport.Embedder, handleError func(error)) search.Search
	SetPluginSearchFn        func(embedder embeddingport.Embedder, handleError func(error))
	OpenNotesFn              func(index note.IndexUpdater) note.Scenarios
	OpenVaultsFn             func(moves note.Move) vaults.Scenarios
	GetRefreshFn             func() vaults.Refresh
	ReadWholeVaultFn         func(ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool) (vaultreading.ReadWholeVault, error)
	StartVaultFn             StartVaultFunc
	PluginServicesFn         func() PluginServices
	PluginCatalogFn          func() PluginCatalog
	PluginCommandsFn         func() plugincommand.Commands
	OpenChooseTabKindFn      func() *tabkind.Choose
	AgentVal                 agentport.Agent
	SetTextVersionListenerFn func(listener textport.VersionListener)
	ReportTextFilesChangedFn func(ctx context.Context, v domain.Vault, paths []string) error
	OpenVersionItemsFn       func() *textversion.VersionItems
	OpenPluginFileFn         func() *file.OpenPluginFile
	OpenTriggersFn           func() *trigger.Triggers
	WatchPluginChangesFn     func() (<-chan struct{}, func())
	GetMediaTypeFn           func() *file.GetMediaType
	OpenChoosePluginPathFn   func(dialog machineport.PathDialog) *settings.ChoosePluginPath
	OpenFileTextFn           func() *textversion.FileText
	OpenPluginDataFn         func() *textversion.PluginData
	OpenPluginVaultFn        func(current func() (domain.Vault, bool), hooks H) (V, error)
}

func (e Runtime[H, V]) GetPluginDir(id string) (string, bool)      { return e.PluginDirFn(id) }
func (e Runtime[H, V]) EmitEvent(event string)                     { e.EmitEventFn(event) }
func (e Runtime[H, V]) GetRegistry() vaultport.VaultRegistry       { return e.RegistryVal }
func (e Runtime[H, V]) GetVaultReaders() vaultport.VaultReaders    { return e.VaultReadersFn() }
func (e Runtime[H, V]) GetVaultWriters() vaultport.VaultWriters    { return e.VaultWritersFn() }
func (e Runtime[H, V]) GetVaultIdentity() vaultport.VaultIdentity  { return e.VaultIdentityFn() }
func (e Runtime[H, V]) GetOpenTabsStore() windowport.OpenTabsStore { return e.OpenTabsStoreFn() }
func (e Runtime[H, V]) GetClock() machineport.Clock                { return e.ClockFn() }
func (e Runtime[H, V]) OpenGetOpeningNote() *query.GetOpeningNote  { return e.OpenGetOpeningNoteFn() }
func (e Runtime[H, V]) OpenGetTypes() *query.GetTypes              { return e.OpenGetTypesFn() }
func (e Runtime[H, V]) OpenGetHeadings() *query.GetHeadings        { return e.OpenGetHeadingsFn() }
func (e Runtime[H, V]) OpenResolveLinks() *query.ResolveLinks      { return e.OpenResolveLinksFn() }
func (e Runtime[H, V]) OpenGetTitles() query.GetTitles             { return e.OpenGetTitlesFn() }
func (e Runtime[H, V]) OpenGetNameMatches() *search.GetNameMatches { return e.OpenGetNameMatchesFn() }
func (e Runtime[H, V]) OpenGetFolderEntries(readers vaultport.VaultReaders) *file.GetFolderEntries {
	return e.OpenGetFolderEntriesFn(readers)
}
func (e Runtime[H, V]) OpenGetFileKinds(readers vaultport.VaultReaders) *file.GetFileKinds {
	return e.OpenGetFileKindsFn(readers)
}
func (e Runtime[H, V]) OpenFocusNote(view windowport.Window) *query.FocusNote {
	return e.OpenFocusNoteFn(view)
}
func (e Runtime[H, V]) GetProblemQueries() problemport.ProblemQueries { return e.ProblemQueriesFn() }
func (e Runtime[H, V]) GetIndexProgress() indexport.IndexProgress     { return e.IndexProgressFn() }
func (e Runtime[H, V]) GetSourceRepository() sourceport.SourceRepository {
	return e.SourceRepositoryFn()
}
func (e Runtime[H, V]) GetSourceQueries() sourceport.SourceQueries   { return e.SourceQueriesFn() }
func (e Runtime[H, V]) GetPassageQueries() sourceport.PassageQueries { return e.PassageQueriesFn() }
func (e Runtime[H, V]) CloseIndex() error                            { return e.CloseIndexFn() }
func (e Runtime[H, V]) OpenGetSettings() *settings.Get               { return e.OpenGetSettingsFn() }
func (e Runtime[H, V]) OpenWriteSettings() *settings.Write           { return e.OpenWriteSettingsFn() }
func (e Runtime[H, V]) OpenWatchSettings() *settings.Watch           { return e.OpenWatchSettingsFn() }
func (e Runtime[H, V]) OpenSecrets() *secret.Secrets                 { return e.OpenSecretsFn() }
func (e Runtime[H, V]) OpenRetryPlugin() *settings.RetryPlugin       { return e.OpenRetryPluginFn() }
func (e Runtime[H, V]) ReportPluginFailures(tasks *run.Tasks)        { e.ReportPluginFailuresFn(tasks) }
func (e Runtime[H, V]) SetPluginWork(tasks *run.Tasks, models *run.Lock) {
	e.SetPluginWorkFn(tasks, models)
}
func (e Runtime[H, V]) IsRebuildingIndex() bool { return e.IsRebuildingIndexVal }
func (e Runtime[H, V]) OpenEmbedders(ctx context.Context, tasks *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error) {
	return e.OpenEmbeddersFn(ctx, tasks)
}
func (e Runtime[H, V]) NewSearch(asking embeddingport.Embedder, handleError func(error)) search.Search {
	return e.NewSearchFn(asking, handleError)
}

// SetPluginSearch lends the plugins the search index and the floors, and does nothing where the
// assembly names no search policy.
func (e Runtime[H, V]) SetPluginSearch(embedder embeddingport.Embedder, handleError func(error)) {
	if e.SetPluginSearchFn == nil {
		return
	}
	e.SetPluginSearchFn(embedder, handleError)
}
func (e Runtime[H, V]) OpenNotes(index note.IndexUpdater) note.Scenarios { return e.OpenNotesFn(index) }
func (e Runtime[H, V]) OpenVaults(moves note.Move) vaults.Scenarios      { return e.OpenVaultsFn(moves) }
func (e Runtime[H, V]) GetRefresh() vaults.Refresh                       { return e.GetRefreshFn() }
func (e Runtime[H, V]) ReadWholeVault(ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool) (vaultreading.ReadWholeVault, error) {
	return e.ReadWholeVaultFn(ctx, embedder, v, rebuild)
}
func (e Runtime[H, V]) StartVault(
	ctx context.Context, v domain.Vault, rebuild bool,
	notified func(paths, assets []string, reload bool),
	handleError window.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vaults.Refresh,
	unwatched error,
) {
	return e.StartVaultFn(ctx, v, rebuild, notified, handleError)
}
func (e Runtime[H, V]) GetPluginServices() PluginServices { return e.PluginServicesFn() }

// GetPluginCatalog is the catalog of plugins, and nil where the assembly names none.
func (e Runtime[H, V]) GetPluginCatalog() PluginCatalog {
	if e.PluginCatalogFn == nil {
		return nil
	}
	return e.PluginCatalogFn()
}
func (e Runtime[H, V]) GetPluginCommands() plugincommand.Commands { return e.PluginCommandsFn() }
func (e Runtime[H, V]) GetAgent() agentport.Agent                 { return e.AgentVal }

// OpenChooseTabKind is the choice of the kind of tab a note opens in, nil where the assembly names
// none.
func (e Runtime[H, V]) OpenChooseTabKind() *tabkind.Choose {
	if e.OpenChooseTabKindFn == nil {
		return nil
	}
	return e.OpenChooseTabKindFn()
}
func (e Runtime[H, V]) SetTextVersionListener(listener textport.VersionListener) {
	e.SetTextVersionListenerFn(listener)
}

// ReportTextFilesChanged reports each changed file whose live text version changed; an assembly with no
// text versions reports none.
func (e Runtime[H, V]) ReportTextFilesChanged(ctx context.Context, v domain.Vault, paths []string) error {
	if e.ReportTextFilesChangedFn == nil {
		return nil
	}
	return e.ReportTextFilesChangedFn(ctx, v, paths)
}
func (e Runtime[H, V]) OpenVersionItems() *textversion.VersionItems { return e.OpenVersionItemsFn() }
func (e Runtime[H, V]) OpenPluginFile() *file.OpenPluginFile        { return e.OpenPluginFileFn() }

func (e Runtime[H, V]) OpenTriggers() *trigger.Triggers {
	if e.OpenTriggersFn == nil {
		return nil
	}
	return e.OpenTriggersFn()
}

// WatchPluginChanges reports each time the host applied settings or forgot the failure of a plugin,
// and never where the installation runs no plugins.
func (e Runtime[H, V]) WatchPluginChanges() (<-chan struct{}, func()) {
	if e.WatchPluginChangesFn == nil {
		return nil, func() {}
	}
	return e.WatchPluginChangesFn()
}

func (e Runtime[H, V]) GetMediaType() *file.GetMediaType { return e.GetMediaTypeFn() }
func (e Runtime[H, V]) OpenChoosePluginPath(dialog machineport.PathDialog) *settings.ChoosePluginPath {
	return e.OpenChoosePluginPathFn(dialog)
}
func (e Runtime[H, V]) OpenFileText() *textversion.FileText     { return e.OpenFileTextFn() }
func (e Runtime[H, V]) OpenPluginData() *textversion.PluginData { return e.OpenPluginDataFn() }
func (e Runtime[H, V]) OpenPluginVault(current func() (domain.Vault, bool), hooks H) (V, error) {
	return e.OpenPluginVaultFn(current, hooks)
}
