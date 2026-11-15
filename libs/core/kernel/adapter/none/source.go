package none

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
)

// SearchSource is the search source of an installation that has none: it describes no model, so nothing is
// embedded.
type SearchSource struct{}

func (SearchSource) Describe() searchsource.Description { return searchsource.Description{} }

func (SearchSource) IndexTexts(context.Context, searchsource.Place, []searchsource.Text) ([][]float32, error) {
	return nil, nil
}

func (SearchSource) SearchChunks(context.Context, searchsource.Place, searchsource.Query) ([]searchsource.Match, error) {
	return nil, nil
}
