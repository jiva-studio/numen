package hostendpoint

import (
	"context"
	"errors"
	"time"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugindependency"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

// Identity tells who made a call and what that plugin may do.
type Identity interface {
	GetPluginIDByToken(token string) (string, bool)
	CheckPluginPermission(pluginID, permission string) error
}

// Leases holds a plugin from being stopped as idle while it works on its own. A lease is taken on the
// process that runs under the token and does not cancel a release of that process; AcquireOwn reports
// whether it took one.
type Leases interface {
	AcquireOwn(pluginID, token string, holder plugin.Holder) bool
	ReleaseOwn(pluginID, token string, holder plugin.Holder)
}

// WorkQueue is the core's shared queue of heavy work and its list of tasks, as the plugins use them.
// Every call names the plugin it is made for.
type WorkQueue interface {
	// EnqueueWork waits for the plugin's turn in the queue and returns it. onWaiting is called where
	// other work holds the queue. A context that ends while waiting takes no turn.
	EnqueueWork(ctx context.Context, pluginID string, isUserStarted bool, onWaiting func()) (*run.Lease, error)
	// ReportTask puts one of the plugin's tasks in the list, or replaces it.
	ReportTask(pluginID string, task plugin.Task)
	// FinishTask takes one of the plugin's tasks out of the list.
	FinishTask(pluginID, taskID string)
	// ClearPlugin takes every task of the plugin out of the list and ends its turns and its waits.
	ClearPlugin(pluginID string)
}

// Account is what a plugin with the identity permission may ask of the identity of this device. The
// seal and sign calls return identity.ErrNoIdentity when the device holds none.
type Account interface {
	// SignAsDevice signs message with the device key under the plugin id.
	SignAsDevice(ctx context.Context, pluginID string, message []byte) ([]byte, error)
	// GetPublicIdentity returns what is public of the identity of this device.
	GetPublicIdentity(ctx context.Context) (identitydomain.Public, error)
	// SealData seals plain for the plugin under the context with the account keys of this device.
	SealData(ctx context.Context, pluginID string, dataContext, plain []byte) ([]byte, error)
	// OpenData opens what SealData sealed for the plugin and the context; failure is identity.ErrCannotOpen.
	OpenData(ctx context.Context, pluginID string, dataContext, sealed []byte) ([]byte, error)
	// OpenSealedBox opens a box sealed to the account box key for the plugin and the context; failure is identity.ErrCannotOpen.
	OpenSealedBox(ctx context.Context, pluginID string, boxContext, sealed []byte) ([]byte, error)
}

// AccountRestore is what a plugin with the identity permission may ask of the restore of an identity that waits
// for the chain of the server. Its calls return identity.ErrNoPendingRestore when none is pending.
type AccountRestore interface {
	// GetPendingRestore returns the public keys of the restore that waits for a chain.
	GetPendingRestore(ctx context.Context) (identitydomain.PendingRestore, error)
	// SignAsRecovery signs with the recovery key of the pending restore.
	SignAsRecovery(ctx context.Context, pluginID string, message []byte) ([]byte, error)
	// CompleteRestore finishes the pending restore with the chain of the server, or with none.
	CompleteRestore(ctx context.Context, pluginID string, chain []byte) error
}

// AccountLink is what a plugin with the identity permission may ask of the link in progress: it carries the blobs of the
// link between the core and the relay of the sync server. Its calls return identity.ErrNoLink when no link is in progress.
type AccountLink interface {
	// GetLinkWork returns what the courier carries; it waits for a change when the version is the one given.
	GetLinkWork(ctx context.Context, sinceVersion uint64, wait time.Duration) (identitydomain.LinkWork, error)
	// SetLinkChannel gives the link the channel the courier made.
	SetLinkChannel(ctx context.Context, id []byte) error
	// PutLinkInbound hands the core a blob of the other side.
	PutLinkInbound(ctx context.Context, blob []byte) error
}

// AccountChain is what a plugin with the identity permission may ask of the chain of this device: the chain the core
// waits to have accepted by the server, and the chain the server holds.
type AccountChain interface {
	// GetChainPublish returns the chain the server must hold before the core goes on.
	GetChainPublish(ctx context.Context) (identitydomain.ChainPublish, error)
	// PutChainPublished tells the core the answer of the server.
	PutChainPublished(ctx context.Context, accepted bool, latest []byte) error
	// AdoptChain gives the core a chain the server holds.
	AdoptChain(ctx context.Context, chain []byte) error
}

// Activation is what the host knows of a plugin that runs: who made a call and what that plugin may do, and the
// leases that hold it from being stopped.
type Activation interface {
	Identity
	Leases
}

// DependencyCalls is what a plugin may ask of the files the host provisions. A host without them answers that it
// has no dependency service.
type DependencyCalls interface {
	// FetchModelFile returns the path of the model file at an address, fetching it when it is not on this machine.
	FetchModelFile(ctx context.Context, address, digest string) (string, error)
	// FetchDeclaredModel returns the path of the model that the manifest of pluginID declares under id.
	FetchDeclaredModel(ctx context.Context, pluginID, id string) (string, error)
}

// errNoDependencyUseCase is a dependency call whose use case the host did not wire.
var errNoDependencyUseCase = errors.New("the host has no dependency service")

// DependencyUseCases serves DependencyCalls with the use cases that fetch a model file and a declared model.
type DependencyUseCases struct {
	Model    *plugindependency.Fetch
	Declared *plugindependency.FetchDeclaredModel
}

func (d DependencyUseCases) FetchModelFile(ctx context.Context, address, digest string) (string, error) {
	if d.Model == nil {
		return "", errNoDependencyUseCase
	}
	return d.Model.Execute(ctx, address, digest)
}

func (d DependencyUseCases) FetchDeclaredModel(ctx context.Context, pluginID, id string) (string, error) {
	if d.Declared == nil {
		return "", errNoDependencyUseCase
	}
	return d.Declared.Execute(ctx, pluginID, id)
}

// Callers is what the host endpoint queries the host for about a call: who made it, what that
// plugin may do, and what the host lends the plugin to work with. Dependencies fetches the files the plugin asks for. Lent holds what exists only while a
// window shows a vault; the tools are read at each call. Files tells which files a plugin takes; without it a plugin takes
// every file. CallCancels holds the cancels of the calls of connections.
type Callers struct {
	Dependencies DependencyCalls
	Activation   Activation
	Lent         *Services
	GetTools     func() PluginTools
	Health       HealthCalls
	Processes    ProcessRelease
	Files        FileTypes
	CallCancels  *searchcancel.Searches
}

// AccountCalls is what a plugin with the identity permission may ask of the identity of this device. Each part is nil
// where the host holds no identity.
type AccountCalls struct {
	Account Account
	Restore AccountRestore
	Link    AccountLink
	Chain   AccountChain
}

func (c AccountCalls) isServed() bool {
	return c.Account != nil || c.Restore != nil || c.Link != nil || c.Chain != nil
}

// LogWriter records a structured log entry reported by a plugin backend.
type LogWriter interface {
	Execute(entry logs.Entry) error
}

// FileTypes tells which files a plugin takes, by the types its manifest declares.
type FileTypes interface {
	IsTakingFile(pluginID, path string, noteTypes []domain.NoteType) bool
}

// ProcessRelease stops the process of a plugin that has finished its work: at once where it holds no
// lease, and else once it holds none. A host without it answers that it does not stop plugins on request.
type ProcessRelease interface {
	ReleaseProcess(ctx context.Context, pluginID string)
}

// HealthReport puts a health item in the list as the plugin's. It returns plugin.ErrInvalidHealth for an
// item the core refuses.
type HealthReport interface {
	Execute(pluginID string, item plugin.HealthItem) error
}

// HealthClear removes the health item of the plugin under the key. It returns plugin.ErrInvalidHealth
// for a key the plugin cannot clear.
type HealthClear interface {
	Execute(pluginID, key string) error
}

// HealthCalls is what a plugin says about whether it works. A host without them answers that it keeps
// no health items.
type HealthCalls interface {
	ReportHealth(pluginID string, item plugin.HealthItem) error
	ClearHealth(pluginID, key string) error
}

// HealthUseCases serves HealthCalls with the use cases that report and clear a health item.
type HealthUseCases struct {
	Report HealthReport
	Clear  HealthClear
}

func (h HealthUseCases) ReportHealth(pluginID string, item plugin.HealthItem) error {
	return h.Report.Execute(pluginID, item)
}

func (h HealthUseCases) ClearHealth(pluginID, key string) error {
	return h.Clear.Execute(pluginID, key)
}

// TextVersions is the text versions of the vaults, as the plugins that call the host work them.
// Each call works the vault it is given.
type TextVersions interface {
	// PutVersion records a version of the plugin, or replaces the one it has under the key.
	PutVersion(ctx context.Context, v domain.Vault, pluginID string, params plugin.TextPut) (domain.TextVersion, error)
	// ListVersions is the versions of a file and which of them is live.
	ListVersions(ctx context.Context, v domain.Vault, path string) (plugin.TextVersionList, error)
	// ReadText is one page of any version's segments, read from its file.
	ReadText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (plugin.TextPage, error)
	// ReadTextRange returns the segments of a version that cover length bytes of its whole text from
	// start, each truncated to the range at whole characters. A length of zero reads to the end.
	ReadTextRange(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, start, length int) (plugin.TextRange, error)
}

// SearchIndex is the search index of the vaults, as the plugins that call the host read it. Each
// call works the vault it is given. A call that names a recipe other than the one of the model in use
// returns plugin.ErrRecipeMismatch, and one on an installation with no model returns plugin.ErrNoModel.
type SearchIndex interface {
	// GetEmbeddingState is the model in use and how far the vault is embedded under it.
	GetEmbeddingState(ctx context.Context, v domain.Vault) (plugin.EmbeddingState, error)
	// GetIndexStats is what the index of the vault holds, with the model in use. The counts are at most
	// one second old and the sizes at most half a minute old.
	GetIndexStats(ctx context.Context, v domain.Vault) (plugin.IndexStats, error)
	// EmbedTexts embeds the texts with the model in use.
	EmbedTexts(ctx context.Context, texts []string) (plugin.EmbeddedTexts, error)
	// SampleSimilarities is the cosine similarity of each query with the small chunks of the vault that
	// carry a vector under the recipe and belong to a source other than the query's note. At most
	// MaxSampledQueries queries are taken, and at most MaxSampledChunks small chunks, in the order of
	// their content hashes. A query longer or shorter than the vectors of the index returns
	// plugin.ErrInvalidQuery.
	SampleSimilarities(ctx context.Context, v domain.Vault, recipe string, queries []plugin.SimilarityQuery) (plugin.SimilaritySample, error)
	// GetBestSimilarity is the cosine similarity of the chunk of the vault nearest the query vector,
	// zero where there is none.
	GetBestSimilarity(ctx context.Context, v domain.Vault, recipe string, query []float32) (float64, error)
}

// TextSearch searches the text of the vaults for the plugins that call the host.
type TextSearch interface {
	// SearchText returns the search results of the vault matching the query, best first, from every
	// source and every text version: by keywords, and by embedding when the installation has a model.
	// An installation with no model searches by keywords only.
	SearchText(ctx context.Context, v domain.Vault, q plugin.SearchResultQuery) ([]domain.SearchResult, error)
	// FindNearestChunks returns the small chunks of the vault nearest the vector under the recipe of the
	// query, nearest first, at or above its floor, else the effective floor of the vault. The recipe
	// rules of SearchIndex apply.
	FindNearestChunks(ctx context.Context, v domain.Vault, q sourcedomain.NearestQuery) ([]searchsource.Match, error)
}

// SearchFloors is the floor a search by meaning applies to the vaults, as the plugins that call the
// host read and set it. The recipe rules of SearchIndex apply.
type SearchFloors interface {
	// GetSearchFloor is the floor a search applies to the vault under the recipe.
	GetSearchFloor(ctx context.Context, v domain.Vault, recipe string) (plugin.SearchFloor, error)
	// SetSearchFloor stores the effective floor of the vault under the recipe, and removes it when the
	// floor is zero.
	SetSearchFloor(ctx context.Context, v domain.Vault, recipe string, floor float64) error
}

// SearchAccess is the search index, the text search and the search floors of the vaults, and the searches
// of the query language, which are nil where the host runs none.
type SearchAccess struct {
	SearchIndex
	TextSearch
	SearchFloors
	Runs *searchqueryusecase.SearchRuns
}

// WindowControl is the main window of the vault, exposed to the plugins that call the host. A path
// with no file in the vault returns fs.ErrNotExist.
type WindowControl interface {
	// FocusNote makes the note at the path the current one and returns it.
	FocusNote(ctx context.Context, path string) (domain.NoteRef, error)
	// FocusFile opens the file at the path at the first range and highlights every range. No
	// ranges opens the file at its beginning.
	FocusFile(ctx context.Context, path string, ranges []domain.ByteRange) error
	// ReadOpenTabs returns the tabs the window last reported, and the active one.
	ReadOpenTabs(ctx context.Context) (windowdomain.OpenTabs, error)
	// ActivateTab makes the open tab with the id the active tab, and returns the open tabs as the
	// window reports them afterwards. An id with no open tab returns fs.ErrNotExist, and a tab the
	// window does not report active in time returns plugin.ErrTabNotShown.
	ActivateTab(ctx context.Context, id string) (windowdomain.OpenTabs, error)
}

// ModelCalls carries out the operations the plugins ask for. The connections come from the rows of the
// calling plugin, and a request names none.
type ModelCalls interface {
	// Call asks the rows in order and sends the data frames to send as they arrive. It returns the done
	// payload, connection.ErrNoConnection, *connection.AllFailedError or *connection.StoppedError where no
	// row answers, the error of send when send fails, and the error of the context when the call is canceled.
	Call(ctx context.Context, callerPluginID string, request modelcall.Request, send modelcall.Send) ([]byte, error)
}

// RowLists lists the rows of the calling plugin and the values of their row fields.
type RowLists interface {
	// ListRows returns the rows of the plugin for the operation in the order of use, without the rows whose
	// connection is missing or whose type is unavailable or does not serve the operation.
	ListRows(callerPluginID, operation string) ([]modelcall.Row, error)
	// ListRowOptions returns the values a row field of a connection among those rows can take. It fails with
	// connection.ErrNotFound for a connection that is not among the rows.
	ListRowOptions(ctx context.Context, callerPluginID, operation, connectionID, field string, options map[string]string) ([]modelcall.Option, error)
}
