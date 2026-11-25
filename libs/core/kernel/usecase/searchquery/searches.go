package searchquery

import (
	"context"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Store keeps the searches in memory.
type Store interface {
	// AddSearch stores a new record and returns the channel closed when the search stops running.
	AddSearch(r searchrun.Record) <-chan struct{}
	// FindSearch returns the record and its stop channel.
	FindSearch(id searchrun.ID) (searchrun.Record, <-chan struct{}, bool)
	// ChangeSearch applies change to the record under the lock of the store and closes the stop channel
	// when change ends the run. It returns false when the store has no record with the id.
	ChangeSearch(id searchrun.ID, change func(*searchrun.Record)) bool
	RemoveSearches(ids []searchrun.ID)
	ListSearches() []searchrun.Record
}

// Cancels holds the cancel of each running search.
type Cancels interface {
	Start(ctx context.Context, id string) (context.Context, func())
	Cancel(id string)
}

// Tasks shows the searches a person or an agent waits for.
type Tasks interface {
	Set(task work.Task)
	Remove(id string)
}

// Searches is the kept searches with the cancels, tasks and policy that go with them.
type Searches struct {
	Store       Store
	Cancels     Cancels
	Tasks       Tasks
	Clock       machineport.Clock
	CreateTimer func(d time.Duration) <-chan time.Time
	Policy      searchrun.Policy
}

func getTaskID(id searchrun.ID) string {
	return "search " + string(id)
}

// Drop cancels the searches and forgets them.
func (s *Searches) Drop(ids []searchrun.ID) {
	for _, id := range ids {
		s.Cancels.Cancel(string(id))
		s.Tasks.Remove(getTaskID(id))
	}
	s.Store.RemoveSearches(ids)
}

// DropExpired drops the searches the policy expires to make room for incoming more, and returns the rest.
func (s *Searches) DropExpired(incoming int) []searchrun.Record {
	records := s.Store.ListSearches()
	expired := searchrun.SelectExpired(records, s.Clock.Now(), s.Policy, incoming)
	s.Drop(expired)
	kept := records[:0]
	for _, r := range records {
		if !slices.Contains(expired, r.ID) {
			kept = append(kept, r)
		}
	}
	return kept
}

// SearchHandle pairs a stored search record with the channel closed when the run stops.
type SearchHandle struct {
	Record      searchrun.Record
	StopChannel <-chan struct{}
}

// Find returns searchrun.ErrNotFound for a search that is unknown, expired, removed or of another owner.
func (s *Searches) Find(owner string, id searchrun.ID) (SearchHandle, error) {
	s.DropExpired(0)
	r, stopped, isFound := s.Store.FindSearch(id)
	if !isFound || r.Start.Owner != owner {
		return SearchHandle{}, searchrun.ErrNotFound
	}
	return SearchHandle{Record: r, StopChannel: stopped}, nil
}

// Touch records that the search was read now.
func (s *Searches) Touch(id searchrun.ID) {
	now := s.Clock.Now()
	s.Store.ChangeSearch(id, func(r *searchrun.Record) { r.AccessedAt = now })
}

func NewSearches(store Store, cancels Cancels, tasks Tasks, clock machineport.Clock, after func(d time.Duration) <-chan time.Time, policy searchrun.Policy) *Searches {
	return &Searches{Store: store, Cancels: cancels, Tasks: tasks, Clock: clock, CreateTimer: after, Policy: policy}
}

// keptSearches is what the use cases of the searches read of the kept searches; an interface keeps the
// fields of Searches out of the use cases that hold it.
type keptSearches interface {
	Parts() *Searches
	Drop(ids []searchrun.ID)
	DropExpired(incoming int) []searchrun.Record
	Find(owner string, id searchrun.ID) (SearchHandle, error)
	Touch(id searchrun.ID)
}

// Parts returns the stores and the policy the kept searches hold.
func (s *Searches) Parts() *Searches { return s }
