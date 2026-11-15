package searchrun

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// searches is what the use cases of this package share of the kept searches.
type searches struct {
	store   SearchStore
	cancels SearchCancels
	tasks   Tasks
	timer   machineport.Timer
	policy  searchrun.Policy
}

func getTaskID(id searchrun.ID) string {
	return "search " + string(id)
}

func (s *searches) drop(ids []searchrun.ID) {
	for _, id := range ids {
		s.cancels.Cancel(string(id))
		s.tasks.Remove(getTaskID(id))
	}
	s.store.RemoveSearches(ids)
}

func (s *searches) dropExpired(incoming int) []searchrun.Record {
	records := s.store.ListSearches()
	expired := searchrun.SelectExpired(records, s.timer.Now(), s.policy, incoming)
	s.drop(expired)
	kept := records[:0]
	for _, r := range records {
		if !slices.Contains(expired, r.ID) {
			kept = append(kept, r)
		}
	}
	return kept
}

func (s *searches) find(owner string, id searchrun.ID) (searchrun.Record, <-chan struct{}, error) {
	s.dropExpired(0)
	r, stopped, isFound := s.store.FindSearch(id)
	if !isFound || r.Start.Owner != owner {
		return searchrun.Record{}, nil, searchrun.ErrNotFound
	}
	return r, stopped, nil
}

func (s *searches) touch(id searchrun.ID) {
	now := s.timer.Now()
	s.store.ChangeSearch(id, func(r *searchrun.Record) { r.AccessedAt = now })
}
