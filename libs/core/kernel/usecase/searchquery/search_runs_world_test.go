package searchquery_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	searchrundomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexgeneration"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchslot"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchstore"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func getTideVault() domain.Vault { return domain.Vault{ID: "tide", Path: "/vaults/tide"} }

// runTimer is the manual clock of a test that tells each wait it is asked for.
type runTimer struct {
	*testsupport.ManualClock
	asked chan time.Duration
}

func (t runTimer) After(d time.Duration) <-chan time.Time {
	fired := t.ManualClock.After(d)
	t.asked <- d
	return fired
}

// runQuery is how the index answers one query text in a test.
type runQuery struct {
	results searchquerydomain.SearchResults
	err     error
	partial *searchquerydomain.SearchResults
	release chan struct{}
}

// runIndex answers the queries of a test and records the order they run in and the ones canceled.
type runIndex struct {
	mu          sync.Mutex
	queries     map[string]*runQuery
	ran         []string
	generations []string
	started     chan string
	canceled    chan string
}

func (x *runIndex) run(ctx context.Context, _ domain.Vault, r searchquerydomain.Request) (searchquerydomain.SearchResults, error) {
	x.mu.Lock()
	q := x.queries[r.Text]
	x.ran = append(x.ran, r.Text)
	x.generations = append(x.generations, r.IndexGeneration)
	x.mu.Unlock()
	if q == nil {
		x.started <- r.Text
		return searchquerydomain.SearchResults{}, nil
	}
	if q.partial != nil {
		r.ReportPartial(*q.partial)
	}
	x.started <- r.Text
	if q.release != nil {
		select {
		case <-q.release:
		case <-ctx.Done():
			x.canceled <- r.Text
			return searchquerydomain.SearchResults{}, ctx.Err()
		}
	}
	return q.results, q.err
}

func (x *runIndex) getRan() []string {
	x.mu.Lock()
	defer x.mu.Unlock()
	return slices.Clone(x.ran)
}

// runTasks is the task list of the window.
type runTasks struct {
	mu    sync.Mutex
	shown map[string]work.Task
}

func (s *runTasks) Set(task work.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shown[task.ID] = task
}

func (s *runTasks) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.shown, id)
}

func (s *runTasks) list() []work.Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []work.Task
	for _, one := range s.shown {
		all = append(all, one)
	}
	return all
}

type runWorld struct {
	runs        searchquery.SearchRuns
	clock       runTimer
	index       *runIndex
	tasks       *runTasks
	generations *indexgeneration.Generations
}

func newRunWorld(t *testing.T, policy searchrundomain.Policy) runWorld {
	t.Helper()
	clock := runTimer{ManualClock: testsupport.NewClock(), asked: make(chan time.Duration, 256)}
	x := &runIndex{queries: map[string]*runQuery{}, started: make(chan string, 64), canceled: make(chan string, 64)}
	shown := &runTasks{shown: map[string]work.Task{}}
	generations := indexgeneration.New("epoch")
	runs := searchquery.NewSearchRuns(searchquery.Deps{
		Store: searchstore.New(), Cancels: searchcancel.NewSearches(clock), Tasks: shown, Clock: clock, CreateTimer: clock.After, Policy: policy,
		Slots: searchslot.New(policy.MaxRunning), Generations: generations, Run: x.run, Names: func(id string) string { return "Plugin " + id },
		GetCatalog: searchquerydomain.GetBuiltinCatalog, MaxResults: 100,
	})
	return runWorld{runs: runs, clock: clock, index: x, tasks: shown, generations: generations}
}

func (w runWorld) hold(text string) *runQuery {
	q := &runQuery{release: make(chan struct{})}
	w.index.mu.Lock()
	w.index.queries[text] = q
	w.index.mu.Unlock()
	return q
}

func (w runWorld) answer(text string, q *runQuery) {
	w.index.mu.Lock()
	w.index.queries[text] = q
	w.index.mu.Unlock()
}

func (w runWorld) start(t *testing.T, owner, text string) searchrundomain.ID {
	t.Helper()
	id, err := w.runs.Start.Execute(t.Context(), searchrundomain.Start{
		Owner: owner, Vault: getTideVault(), Query: text, Priority: modelcall.PriorityInteractive,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (w runWorld) readFinalPage(t *testing.T, owner string, id searchrundomain.ID) searchquery.Page {
	t.Helper()
	done := make(chan searchquery.Page, 1)
	go func() {
		page, err := w.runs.Get.Execute(context.WithoutCancel(t.Context()), searchquery.Read{Owner: owner, ID: id, Wait: 30 * time.Second})
		if err != nil {
			t.Error(err)
		}
		done <- page
	}()
	select {
	case page := <-done:
		return page
	case <-time.After(10 * time.Second):
		t.Fatal("the search did not stop")
	}
	return searchquery.Page{}
}

func getResults(paths ...string) searchquerydomain.SearchResults {
	var results searchquerydomain.SearchResults
	for _, p := range paths {
		results.Results = append(results.Results, searchquerydomain.SearchResult{Document: searchquerydomain.Document{Path: p}})
	}
	return results
}

func getPagePaths(page searchquery.Page) []string {
	var paths []string
	for _, r := range page.Record.Results.Results {
		paths = append(paths, r.Document.Path)
	}
	return paths
}
