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
	// SearchChunks returns the chunks of the place that answer the query, best first.
	SearchChunks(ctx context.Context, place searchsource.Place, q searchsource.Query) ([]searchsource.Match, error)
}
