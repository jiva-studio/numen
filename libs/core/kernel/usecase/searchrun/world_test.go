package searchrun_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	searchrundomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/indexgeneration"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/searchstore"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchrun"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func getTideVault() domain.Vault { return domain.Vault{ID: "tide", Path: "/vaults/tide"} }

// timer is the manual clock of a test that tells each wait it is asked for.
type timer struct {
	*testsupport.ManualClock
	asked chan time.Duration
}

func (t timer) After(d time.Duration) <-chan time.Time {
	fired := t.ManualClock.After(d)
	t.asked <- d
	return fired
}

// query is how the index answers one query text in a test.
type query struct {
	results searchquery.SearchResults
	err     error
	partial *searchquery.SearchResults
	release chan struct{}
}

// index answers the queries of a test and records the order they run in and the ones canceled.
type index struct {
	mu          sync.Mutex
	queries     map[string]*query
	ran         []string
	generations []string
	started     chan string
	canceled    chan string
}

func (x *index) run(ctx context.Context, _ domain.Vault, r searchquery.Request) (searchquery.SearchResults, error) {
	x.mu.Lock()
	q := x.queries[r.Text]
	x.ran = append(x.ran, r.Text)
	x.generations = append(x.generations, r.IndexGeneration)
	x.mu.Unlock()
	if q == nil {
		x.started <- r.Text
		return searchquery.SearchResults{}, nil
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
			return searchquery.SearchResults{}, ctx.Err()
		}
	}
	return q.results, q.err
}

func (x *index) getRan() []string {
	x.mu.Lock()
	defer x.mu.Unlock()
	return slices.Clone(x.ran)
}

// tasks is the task list of the window.
type tasks struct {
	mu    sync.Mutex
	shown map[string]work.Task
}

func (s *tasks) Set(task work.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shown[task.ID] = task
}

func (s *tasks) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.shown, id)
}

func (s *tasks) list() []work.Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []work.Task
	for _, one := range s.shown {
		all = append(all, one)
	}
	return all
}

type world struct {
	runs        searchrun.SearchRuns
	clock       timer
	index       *index
	tasks       *tasks
	generations *indexgeneration.Generations
}

func newWorld(t *testing.T, policy searchrundomain.Policy) world {
	t.Helper()
	clock := timer{ManualClock: testsupport.NewClock(), asked: make(chan time.Duration, 256)}
	x := &index{queries: map[string]*query{}, started: make(chan string, 64), canceled: make(chan string, 64)}
	shown := &tasks{shown: map[string]work.Task{}}
	generations := indexgeneration.New("epoch")
	runs := searchrun.NewSearchRuns(searchrun.Deps{
		Store: searchstore.New(), Generations: generations, Run: x.run, Cancels: searchcancel.NewSearches(clock),
		Tasks: shown, Names: func(id string) string { return "Plugin " + id }, Timer: clock,
		Catalog: searchquery.GetBuiltinCatalog, Policy: policy, MaxResults: 100,
	})
	return world{runs: runs, clock: clock, index: x, tasks: shown, generations: generations}
}

func (w world) hold(text string) *query {
	q := &query{release: make(chan struct{})}
	w.index.mu.Lock()
	w.index.queries[text] = q
	w.index.mu.Unlock()
	return q
}

func (w world) answer(text string, q *query) {
	w.index.mu.Lock()
	w.index.queries[text] = q
	w.index.mu.Unlock()
}

func (w world) start(t *testing.T, owner, text string) searchrundomain.ID {
	t.Helper()
	id, err := w.runs.Start.Execute(t.Context(), searchrundomain.Start{
		Owner: owner, Vault: getTideVault(), Query: text, Priority: modelcall.PriorityInteractive,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (w world) readFinalPage(t *testing.T, owner string, id searchrundomain.ID) searchrun.Page {
	t.Helper()
	done := make(chan searchrun.Page, 1)
	go func() {
		page, err := w.runs.Get.Execute(context.WithoutCancel(t.Context()), searchrun.Read{Owner: owner, ID: id, Wait: 30 * time.Second})
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
	return searchrun.Page{}
}

func getResults(paths ...string) searchquery.SearchResults {
	var results searchquery.SearchResults
	for _, p := range paths {
		results.Results = append(results.Results, searchquery.SearchResult{Document: searchquery.Document{Path: p}})
	}
	return results
}

func getPaths(page searchrun.Page) []string {
	var paths []string
	for _, r := range page.Record.Results.Results {
		paths = append(paths, r.Document.Path)
	}
	return paths
}
