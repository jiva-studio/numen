package searchrun

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/searchkept"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/searchslot"
)

// Deps is what the searches read. MaxResults is the most results one search holds.
type Deps struct {
	Kept        *searchkept.Searches
	Generations IndexGenerations
	Run         RunQuery
	Names       PluginNames
	Catalog     func() searchquery.Catalog
	MaxResults  int
}

// SearchRuns is the use cases of the searches plugins start, sharing one count of running searches.
type SearchRuns struct {
	Start     *StartSearch
	Get       *GetSearch
	Remove    *RemoveSearch
	Functions *ListSearchFunctions
}

// NewSearchRuns creates the use cases of the searches over deps.
func NewSearchRuns(deps Deps) SearchRuns {
	return SearchRuns{
		Start: &StartSearch{
			kept: deps.Kept, generations: deps.Generations, run: deps.Run, names: deps.Names,
			slots: searchslot.New(deps.Kept.Policy.MaxRunning), maxResults: deps.MaxResults,
		},
		Get:       &GetSearch{kept: deps.Kept, generations: deps.Generations},
		Remove:    &RemoveSearch{kept: deps.Kept},
		Functions: &ListSearchFunctions{catalog: deps.Catalog},
	}
}
