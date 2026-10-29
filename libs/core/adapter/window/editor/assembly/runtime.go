package assembly

import (
	"context"

	agentport "github.com/jiva-studio/numen/libs/core/usecase/completion/agentcontract"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	machineport "github.com/jiva-studio/numen/libs/core/usecase/file/machinecontract"
	sourceport "github.com/jiva-studio/numen/libs/core/usecase/filetext/sourcecontract"
	"github.com/jiva-studio/numen/libs/core/usecase/floor"
	indexport "github.com/jiva-studio/numen/libs/core/usecase/indexing/indexcontract"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	noteport "github.com/jiva-studio/numen/libs/core/usecase/note/notecontract"
	problemport "github.com/jiva-studio/numen/libs/core/usecase/note/problemcontract"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	textport "github.com/jiva-studio/numen/libs/core/usecase/textversion/textcontract"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
	windowport "github.com/jiva-studio/numen/libs/core/usecase/vault/windowcontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/usecase/vectors/embeddingcontract"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// StartVaultFunc is the signature for starting a vault session.
type StartVaultFunc func(
	ctx context.Context, v domain.Vault, rebuild bool,
	onSync func(paths, assets []string, reload bool),
	handleError windowport.ErrorHandler,
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
	NoteQueriesFn            func() noteport.NoteQueries
	LinkQueriesFn            func() noteport.LinkQueries
	ProblemQueriesFn         func() problemport.ProblemQueries
	IndexProgressFn          func() indexport.IndexProgress
	SourceRepositoryFn       func() sourceport.SourceRepository
	SourceQueriesFn          func() sourceport.SourceQueries
	PassageQueriesFn         func() sourceport.PassageQueries
	CloseIndexFn             func() error
	OpenGetSettingsFn        func() *settings.Get
	OpenWriteSettingsFn      func() *settings.Write
	OpenWritePluginSecretFn  func() *settings.WriteSecret
	OpenDeletePluginSecretFn func() *settings.DeleteSecret
	OpenRetryPluginFn        func() *settings.RetryPlugin
	ReportPluginFailuresFn   func(tasks *run.Tasks)
	SetPluginWorkFn          func(tasks *run.Tasks, models *run.Lock)
	IsRebuildingIndexVal     bool
	OpenEmbeddersFn          func(ctx context.Context, tasks *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error)
	NewSearchFn              func(asking embeddingport.Embedder, handleError func(error)) search.Search
	OpenFloorsFn             func(embedder embeddingport.Embedder, handleError func(error)) floor.Scenarios
	OpenNotesFn              func(index note.Levels) note.Scenarios
	OpenVaultsFn             func(moves note.Move) vaults.Scenarios
	GetRefreshFn             func() vaults.Refresh
	ReadWholeVaultFn         func(ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool) (vaults.ReadWholeVault, error)
	StartVaultFn             StartVaultFunc
	PluginServicesFn         func() PluginServices
	PluginCommandsFn         func() plugincommand.Commands
	AgentVal                 agentport.Agent
	SetTextVersionListenerFn func(listener textport.VersionListener)
	OpenFileMenuFn           func() *textversion.FileMenu
	OpenProxyFn              func() *file.OpenProxy
	GetMediaTypeFn           func() *file.GetMediaType
	OpenChoosePluginPathFn   func(dialog machineport.PathDialog) *settings.ChoosePluginPath
	OpenFileTextFn           func() *textversion.FileText
	OpenPluginDataFn         func() *textversion.PluginData
	OpenPluginVaultFn        func(current func() (domain.Vault, bool), hooks H) (V, error)
}

func (e Runtime[H, V]) GetPluginDir(id string) (string, bool)         { return e.PluginDirFn(id) }
func (e Runtime[H, V]) EmitEvent(event string)                        { e.EmitEventFn(event) }
func (e Runtime[H, V]) GetRegistry() vaultport.VaultRegistry          { return e.RegistryVal }
func (e Runtime[H, V]) GetVaultReaders() vaultport.VaultReaders       { return e.VaultReadersFn() }
func (e Runtime[H, V]) GetVaultWriters() vaultport.VaultWriters       { return e.VaultWritersFn() }
func (e Runtime[H, V]) GetVaultIdentity() vaultport.VaultIdentity     { return e.VaultIdentityFn() }
func (e Runtime[H, V]) GetOpenTabsStore() windowport.OpenTabsStore    { return e.OpenTabsStoreFn() }
func (e Runtime[H, V]) GetClock() machineport.Clock                   { return e.ClockFn() }
func (e Runtime[H, V]) GetNoteQueries() noteport.NoteQueries          { return e.NoteQueriesFn() }
func (e Runtime[H, V]) GetLinkQueries() noteport.LinkQueries          { return e.LinkQueriesFn() }
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
func (e Runtime[H, V]) OpenWritePluginSecret() *settings.WriteSecret {
	return e.OpenWritePluginSecretFn()
}
func (e Runtime[H, V]) OpenDeletePluginSecret() *settings.DeleteSecret {
	return e.OpenDeletePluginSecretFn()
}
func (e Runtime[H, V]) OpenRetryPlugin() *settings.RetryPlugin { return e.OpenRetryPluginFn() }
func (e Runtime[H, V]) ReportPluginFailures(tasks *run.Tasks)  { e.ReportPluginFailuresFn(tasks) }
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

// OpenFloors is the floor use cases, and none where the assembly names no floors.
func (e Runtime[H, V]) OpenFloors(embedder embeddingport.Embedder, handleError func(error)) floor.Scenarios {
	if e.OpenFloorsFn == nil {
		return floor.Scenarios{}
	}
	return e.OpenFloorsFn(embedder, handleError)
}
func (e Runtime[H, V]) OpenNotes(index note.Levels) note.Scenarios  { return e.OpenNotesFn(index) }
func (e Runtime[H, V]) OpenVaults(moves note.Move) vaults.Scenarios { return e.OpenVaultsFn(moves) }
func (e Runtime[H, V]) GetRefresh() vaults.Refresh                  { return e.GetRefreshFn() }
func (e Runtime[H, V]) ReadWholeVault(ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool) (vaults.ReadWholeVault, error) {
	return e.ReadWholeVaultFn(ctx, embedder, v, rebuild)
}
func (e Runtime[H, V]) StartVault(
	ctx context.Context, v domain.Vault, rebuild bool,
	notified func(paths, assets []string, reload bool),
	handleError windowport.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vaults.Refresh,
	unwatched error,
) {
	return e.StartVaultFn(ctx, v, rebuild, notified, handleError)
}
func (e Runtime[H, V]) GetPluginServices() PluginServices         { return e.PluginServicesFn() }
func (e Runtime[H, V]) GetPluginCommands() plugincommand.Commands { return e.PluginCommandsFn() }
func (e Runtime[H, V]) GetAgent() agentport.Agent                 { return e.AgentVal }
func (e Runtime[H, V]) SetTextVersionListener(listener textport.VersionListener) {
	e.SetTextVersionListenerFn(listener)
}
func (e Runtime[H, V]) OpenFileMenu() *textversion.FileMenu { return e.OpenFileMenuFn() }
func (e Runtime[H, V]) OpenProxy() *file.OpenProxy          { return e.OpenProxyFn() }
func (e Runtime[H, V]) GetMediaType() *file.GetMediaType    { return e.GetMediaTypeFn() }
func (e Runtime[H, V]) OpenChoosePluginPath(dialog machineport.PathDialog) *settings.ChoosePluginPath {
	return e.OpenChoosePluginPathFn(dialog)
}
func (e Runtime[H, V]) OpenFileText() *textversion.FileText     { return e.OpenFileTextFn() }
func (e Runtime[H, V]) OpenPluginData() *textversion.PluginData { return e.OpenPluginDataFn() }
func (e Runtime[H, V]) OpenPluginVault(current func() (domain.Vault, bool), hooks H) (V, error) {
	return e.OpenPluginVaultFn(current, hooks)
}
