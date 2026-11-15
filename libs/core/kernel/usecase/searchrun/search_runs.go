package searchrun

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Deps is what the searches read. MaxResults is the most results one search holds.
type Deps struct {
	Store       SearchStore
	Generations IndexGenerations
	Run         RunQuery
	Cancels     SearchCancels
	Tasks       Tasks
	Names       PluginNames
	Timer       machineport.Timer
	Catalog     func() searchquery.Catalog
	Policy      searchrun.Policy
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
	kept := &searches{store: deps.Store, cancels: deps.Cancels, tasks: deps.Tasks, timer: deps.Timer, policy: deps.Policy}
	return SearchRuns{
		Start: &StartSearch{
			searches: kept, generations: deps.Generations, run: deps.Run, names: deps.Names,
			slots: newSlots(deps.Policy.MaxRunning), maxResults: deps.MaxResults,
		},
		Get:       &GetSearch{searches: kept, generations: deps.Generations},
		Remove:    &RemoveSearch{searches: kept},
		Functions: &ListSearchFunctions{catalog: deps.Catalog},
	}
}
