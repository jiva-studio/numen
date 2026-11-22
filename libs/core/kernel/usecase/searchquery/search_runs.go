package searchquery

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Deps is what the searches read. MaxResults is the most results one search holds, and Slots is shared by
// every search that runs.
type Deps struct {
	Store       Store
	Cancels     Cancels
	Tasks       Tasks
	Clock       machineport.Clock
	After       func(d time.Duration) <-chan time.Time
	Policy      searchrun.Policy
	Slots       SearchSlots
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
	kept := &Searches{Store: deps.Store, Cancels: deps.Cancels, Tasks: deps.Tasks, Clock: deps.Clock, After: deps.After, Policy: deps.Policy}
	return SearchRuns{
		Start: &StartSearch{
			kept: kept, generations: deps.Generations, run: deps.Run, names: deps.Names,
			slots: deps.Slots, maxResults: deps.MaxResults,
		},
		Get:       &GetSearch{kept: kept, generations: deps.Generations},
		Remove:    &RemoveSearch{kept: kept},
		Functions: &ListSearchFunctions{catalog: deps.Catalog},
	}
}
