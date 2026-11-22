// Package searchrun runs the searches of the query language that plugins start, keeps their results for a
// while, and answers pages of them to the plugin that started each.
package searchquery

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// IndexGenerations gives the generation of the index of a vault: a text that changes whenever indexing
// changes the index of the vault, and differs between two starts of the core.
type IndexGenerations interface {
	GetGeneration(v domain.VaultID) string
}

// RunQuery runs a query of the language over a vault.
type RunQuery func(ctx context.Context, v domain.Vault, r searchquery.Request) (searchquery.SearchResults, error)

// PluginNames returns the manifest name of a plugin, which its search tasks show.
type PluginNames func(pluginID string) string

// SearchSlots limits how many searches run at once and queues the others by priority.
type SearchSlots interface {
	Acquire(ctx context.Context, priority modelcall.Priority) error
	Release()
}
