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
	store   Store
	cancels Cancels
	tasks   Tasks
	clock   machineport.Clock
	after   func(d time.Duration) <-chan time.Time
	policy  searchrun.Policy
}

func (s *Searches) Store() Store                                { return s.store }
func (s *Searches) Cancels() Cancels                            { return s.cancels }
func (s *Searches) Tasks() Tasks                                { return s.tasks }
func (s *Searches) Clock() machineport.Clock                    { return s.clock }
func (s *Searches) After() func(time.Duration) <-chan time.Time { return s.after }
func (s *Searches) Policy() searchrun.Policy                    { return s.policy }

func getTaskID(id searchrun.ID) string {
	return "search " + string(id)
}

// Drop cancels the searches and forgets them.
func (s *Searches) Drop(ids []searchrun.ID) {
	for _, id := range ids {
		s.cancels.Cancel(string(id))
		s.tasks.Remove(getTaskID(id))
	}
	s.store.RemoveSearches(ids)
}

// DropExpired drops the searches the policy expires to make room for incoming more, and returns the rest.
func (s *Searches) DropExpired(incoming int) []searchrun.Record {
	records := s.store.ListSearches()
	expired := searchrun.SelectExpired(records, s.clock.Now(), s.policy, incoming)
	s.Drop(expired)
	kept := records[:0]
	for _, r := range records {
		if !slices.Contains(expired, r.ID) {
			kept = append(kept, r)
		}
	}
	return kept
}

// Find returns searchrun.ErrNotFound for a search that is unknown, expired, removed or of another owner.
func (s *Searches) Find(owner string, id searchrun.ID) (searchrun.Record, <-chan struct{}, error) {
	s.DropExpired(0)
	r, stopped, isFound := s.store.FindSearch(id)
	if !isFound || r.Start.Owner != owner {
		return searchrun.Record{}, nil, searchrun.ErrNotFound
	}
	return r, stopped, nil
}

// Touch records that the search was read now.
func (s *Searches) Touch(id searchrun.ID) {
	now := s.clock.Now()
	s.store.ChangeSearch(id, func(r *searchrun.Record) { r.AccessedAt = now })
}

func NewSearches(store Store, cancels Cancels, tasks Tasks, clock machineport.Clock, after func(d time.Duration) <-chan time.Time, policy searchrun.Policy) *Searches {
	return &Searches{store: store, cancels: cancels, tasks: tasks, clock: clock, after: after, policy: policy}
}

// keptSearches is what the use cases of the searches read of the kept searches; an interface keeps the
// fields of Searches out of the use cases that hold it.
type keptSearches interface {
	Store() Store
	Cancels() Cancels
	Tasks() Tasks
	Clock() machineport.Clock
	After() func(d time.Duration) <-chan time.Time
	Policy() searchrun.Policy
	Drop(ids []searchrun.ID)
	DropExpired(incoming int) []searchrun.Record
	Find(owner string, id searchrun.ID) (searchrun.Record, <-chan struct{}, error)
	Touch(id searchrun.ID)
}
