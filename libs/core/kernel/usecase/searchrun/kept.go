package searchrun

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Kept is the kept searches with the cancels, tasks and policy that go with them.
type Kept struct {
	Store   SearchStore
	Cancels SearchCancels
	Tasks   Tasks
	Timer   machineport.Timer
	Policy  searchrun.Policy
}

func getTaskID(id searchrun.ID) string {
	return "search " + string(id)
}

// drop cancels the searches and forgets them.
func (s *Kept) drop(ids []searchrun.ID) {
	for _, id := range ids {
		s.Cancels.Cancel(string(id))
		s.Tasks.Remove(getTaskID(id))
	}
	s.Store.RemoveSearches(ids)
}

// dropExpired drops the searches the policy expires to make room for incoming more, and returns the rest.
func (s *Kept) dropExpired(incoming int) []searchrun.Record {
	records := s.Store.ListSearches()
	expired := searchrun.SelectExpired(records, s.Timer.Now(), s.Policy, incoming)
	s.drop(expired)
	kept := records[:0]
	for _, r := range records {
		if !slices.Contains(expired, r.ID) {
			kept = append(kept, r)
		}
	}
	return kept
}

// find returns searchrun.ErrNotFound for a search that is unknown, expired, removed or of another owner.
func (s *Kept) find(owner string, id searchrun.ID) (searchrun.Record, <-chan struct{}, error) {
	s.dropExpired(0)
	r, stopped, isFound := s.Store.FindSearch(id)
	if !isFound || r.Start.Owner != owner {
		return searchrun.Record{}, nil, searchrun.ErrNotFound
	}
	return r, stopped, nil
}

// touch records that the search was read now.
func (s *Kept) touch(id searchrun.ID) {
	now := s.Timer.Now()
	s.Store.ChangeSearch(id, func(r *searchrun.Record) { r.AccessedAt = now })
}
