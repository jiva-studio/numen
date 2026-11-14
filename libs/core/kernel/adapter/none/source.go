package none

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// SourceBacklog is the backlog of an index that has no source to cut again.
type SourceBacklog struct{}

func (SourceBacklog) GetUnchunkedSources(context.Context, domain.VaultID, domain.SourceKind, int) ([]string, error) {
	return nil, nil
}

func (SourceBacklog) GetSourcesWithOtherRecipe(context.Context, domain.VaultID, domain.SourceKind, []string, int) ([]string, error) {
	return nil, nil
}

// SearchSource is the search source of an installation that has none: it describes no model, so nothing is
// embedded.
type SearchSource struct{}

func (SearchSource) Describe() searchsource.Description { return searchsource.Description{} }

func (SearchSource) IndexTexts(context.Context, searchsource.Place, []searchsource.Text) ([][]float32, error) {
	return nil, nil
}

func (SearchSource) SearchPassages(context.Context, searchsource.Place, searchsource.Query) ([]searchsource.Match, error) {
	return nil, nil
}
