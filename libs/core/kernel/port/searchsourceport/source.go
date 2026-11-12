package searchsourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
)

// Source is a search source a plugin provides: it turns the texts the core hands it into the vectors the
// core keeps in its place, and answers a query with the chunks of that place. The core does not know how
// it makes a vector.
type Source interface {
	// Describe is the description every vector this source returns belongs to.
	Describe() searchsource.Description
	// IndexTexts returns one vector per text, in the order the texts arrived, of the width the description
	// names. Any number of texts may be passed.
	IndexTexts(ctx context.Context, place searchsource.Place, texts []searchsource.Text) ([][]float32, error)
	// SearchPassages returns the chunks of the place that answer the query, best first.
	SearchPassages(ctx context.Context, place searchsource.Place, q searchsource.Query) ([]searchsource.Match, error)
	// DescribeCoverage returns what a person reads about how far the source has indexed the vault of the
	// place, and empty where there is nothing to say.
	DescribeCoverage(ctx context.Context, place searchsource.Place, counts searchsource.Counts) (string, error)
	// Concurrency is how many calls of IndexTexts the source answers at once, and one for a source that
	// answers them one at a time.
	Concurrency() int
}

// QueryEmbedder makes the vectors of queries the way a source compares a query with its place. A plugin
// that calibrates the floor of a vault measures it with them.
type QueryEmbedder interface {
	// EmbedQueries returns one vector per query, in the order the queries arrived.
	EmbedQueries(ctx context.Context, queries []string) ([][]float32, error)
}

// FloorAdvisor reports the similarity a passage starts out having to reach to answer a query under the
// source. The hint is not part of the identity and is absent from the recipe.
type FloorAdvisor interface {
	// GetFloorHint returns the starting floor, and zero where the source has none.
	GetFloorHint(ctx context.Context) float64
}

// DescriptionWatcher reports when the description of a source changes.
type DescriptionWatcher interface {
	// OnDescriptionChange registers fn to run each time the description changes.
	OnDescriptionChange(fn func())
}

// WatchedSource is a Source whose description can change while the window is open, as it does when the
// settings of its plugin change.
type WatchedSource interface {
	Source
	DescriptionWatcher
}
