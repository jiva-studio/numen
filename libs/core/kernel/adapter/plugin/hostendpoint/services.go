package hostendpoint

import (
	"context"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/completion"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

// Identity tells who made a call and what that plugin may do.
type Identity interface {
	GetPluginIDByToken(token string) (string, bool)
	CheckPluginPermission(pluginID, permission string) error
}

// Leases holds a plugin from being stopped as idle while it works.
type Leases interface {
	Acquire(pluginID string)
	Release(pluginID string)
}

// Completer answers a question of a plugin with text, from the default agent of the settings.
type Completer interface {
	// Complete answers the question of the plugin with the given id. A question no agent answers
	// fails with completion.ErrNoRoute, and an unreachable agent with ErrUnavailable.
	Complete(ctx context.Context, pluginID string, question completion.Question) (string, error)
}

// CompletionSource lends the completer of the host.
type CompletionSource interface {
	// GetCompleter returns the completer, or nil when the host has none.
	GetCompleter() Completer
}

// VaultSource lends the vault the host works.
type VaultSource interface {
	// GetPluginVault returns the open vault, or nil when the host has none.
	GetPluginVault() *vault.Access
}

// TextSource lends the text versions of the vaults.
type TextSource interface {
	// GetPluginTexts returns the text versions, or nil when the host has none.
	GetPluginTexts() TextVersions
}

// SearchSource lends the search of the vaults.
type SearchSource interface {
	// GetPluginSearch returns the search, or nil when the host has none.
	GetPluginSearch() *SearchAccess
}

// WindowSource lends the window of the vault and the list of vaults it shows.
type WindowSource interface {
	// GetPluginWindow returns the main window of the vault, or nil when the host has no window.
	GetPluginWindow() WindowControl
	// GetPluginVaultList returns the list of vaults plugins change, or nil when the host has no window.
	GetPluginVaultList() vault.List
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

// WorkSource lends the queue of heavy work and the list of tasks.
type WorkSource interface {
	// GetPluginWork returns the queue, or nil when the host has none.
	GetPluginWork() WorkQueue
}

// DependencySource lends what provisions the dependencies of the plugins.
type DependencySource interface {
	// GetDependencies returns the provisioner, or nil when the host has none.
	GetDependencies() Dependencies
}

// ToolSource lends the tools the plugins offer.
type ToolSource interface {
	GetPluginTools() PluginTools
}

// Callers is what the host endpoint queries the host for about a call: who made it, what that
// plugin may do, and what the host lends the plugin to work with.
type Callers struct {
	Identity     Identity
	Leases       Leases
	Completion   CompletionSource
	Vault        VaultSource
	Texts        TextSource
	Search       SearchSource
	Window       WindowSource
	Work         WorkSource
	Dependencies DependencySource
	Tools        ToolSource
}

// TextVersions is the text versions of the vaults, as the plugins that call the host work them.
// Each call works the vault it is given.
type TextVersions interface {
	// PutVersion records a version of the plugin, or replaces the one it has under the key.
	PutVersion(ctx context.Context, v domain.Vault, pluginID string, params plugin.TextPut) (domain.TextVersion, error)
	// RemoveVersion removes the record of a version of the plugin.
	RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error
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
	// MaxSampledQueries queries are taken, and at most MaxSampledPassages small chunks, in the order of
	// their content hashes. A query longer or shorter than the vectors of the index returns
	// plugin.ErrInvalidQuery.
	SampleSimilarities(ctx context.Context, v domain.Vault, recipe string, queries []plugin.SimilarityQuery) (plugin.SimilaritySample, error)
	// GetBestSimilarity is the cosine similarity of the passage of the vault nearest the query vector,
	// zero where there is none.
	GetBestSimilarity(ctx context.Context, v domain.Vault, recipe string, query []float32) (float64, error)
}

// PassageSearch searches the text of the vaults for the plugins that call the host.
type PassageSearch interface {
	// SearchPassages returns the passages of the vault matching the query, best first, from every
	// source and every text version: by keywords, and by embedding when the installation has a model.
	// An installation with no model searches by keywords only.
	SearchPassages(ctx context.Context, v domain.Vault, q plugin.PassageQuery) ([]domain.Passage, error)
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

// SearchAccess is the search index, the passage search and the search floors of the vaults.
type SearchAccess struct {
	SearchIndex
	PassageSearch
	SearchFloors
}

// WindowControl is the main window of the vault, exposed to the plugins that call the host. A path
// with no file in the vault returns fs.ErrNotExist.
type WindowControl interface {
	// FocusNote makes the note at the path the current one and returns it.
	FocusNote(ctx context.Context, path string) (domain.NoteRef, error)
	// FocusPassages opens the file at the path at the first passage and highlights every passage. No
	// passages opens the file at its beginning.
	FocusPassages(ctx context.Context, path string, passages []domain.ByteRange) error
	// ReadOpenTabs returns the tabs the window last reported, and the active one.
	ReadOpenTabs(ctx context.Context) (windowdomain.OpenTabs, error)
	// ActivateTab makes the open tab with the id the active tab, and returns the open tabs as the
	// window reports them afterwards. An id with no open tab returns fs.ErrNotExist, and a tab the
	// window does not report active in time returns plugin.ErrTabNotShown.
	ActivateTab(ctx context.Context, id string) (windowdomain.OpenTabs, error)
}

// DependencySourceFunc lends the provisioner the function returns, or nil.
type DependencySourceFunc func() Dependencies

// GetDependencies returns the provisioner, or nil when the host has none.
func (f DependencySourceFunc) GetDependencies() Dependencies { return f() }

// ToolSourceFunc lends the tools the function returns.
type ToolSourceFunc func() PluginTools

func (f ToolSourceFunc) GetPluginTools() PluginTools { return f() }
