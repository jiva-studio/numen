// Package searchrun runs the searches of the query language that plugins start, keeps their results for a
// while, and answers pages of them to the plugin that started each.
package searchrun

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// SearchStore keeps the searches in memory.
type SearchStore interface {
	// AddSearch stores a new record and returns the channel closed when the search stops running.
	AddSearch(r searchrun.Record) <-chan struct{}
	// FindSearch returns the record and its stop channel.
	FindSearch(id searchrun.ID) (searchrun.Record, <-chan struct{}, bool)
	// ChangeSearch applies change to the record under the lock of the store and closes the stop channel
	// when change ends the run. It returns false when the store has no record with the id.
	ChangeSearch(id searchrun.ID, change func(*searchrun.Record)) bool
	RemoveSearches(ids []searchrun.ID)
	ListSearches() []searchrun.Record
}

// IndexGenerations gives the generation of the index of a vault: a text that changes whenever indexing
// changes the index of the vault, and differs between two starts of the core.
type IndexGenerations interface {
	GetGeneration(v domain.VaultID) string
}

// RunQuery runs a query of the language over a vault.
type RunQuery func(ctx context.Context, v domain.Vault, r searchquery.Request) (searchquery.SearchResults, error)

// SearchCancels holds the cancel of each running search.
type SearchCancels interface {
	Start(ctx context.Context, id string) (context.Context, func())
	Cancel(id string)
}

// Tasks shows the searches a person or an agent waits for.
type Tasks interface {
	Set(task work.Task)
	Remove(id string)
}

// PluginNames returns the manifest name of a plugin, which its search tasks show.
type PluginNames func(pluginID string) string
