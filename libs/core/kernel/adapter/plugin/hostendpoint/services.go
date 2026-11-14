package hostendpoint

import (
	"context"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
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

// Callers is what the host endpoint queries the host for about a call: who made it, what that
// plugin may do, and what the host lends the plugin to work with. Lent holds what exists only while a
// window shows a vault; the tools are read at each call.
type Callers struct {
	Identity  Identity
	Leases    Leases
	Lent      *Services
	Tools     func() PluginTools
	Health    HealthCalls
	Processes ProcessRelease
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
type HealthCalls struct {
	Report HealthReport
	Clear  HealthClear
}

// TextVersions is the text versions of the vaults, as the plugins that call the host work them.
// Each call works the vault it is given.
type TextVersions interface {
	// PutVersion records a version of the plugin, or replaces the one it has under the key.
	PutVersion(ctx context.Context, v domain.Vault, pluginID string, params plugin.TextPut) (domain.TextVersion, error)
	// ListVersions is the versions of a file and which of them is live.
	ListVersions(ctx context.Context, v domain.Vault, path string) (plugin.TextListing, error)
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

// SearchAccess is the search index, the text search and the search floors of the vaults.
type SearchAccess struct {
	SearchIndex
	TextSearch
	SearchFloors
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
