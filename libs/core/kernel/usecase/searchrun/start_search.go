package searchrun

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/ulid"
)

const (
	taskTitle     = "Search"
	taskAboutSize = 80
)

// StartSearch starts a search, or returns the id of an equal search of the same plugin that is still
// running.
type StartSearch struct {
	*searches
	generations IndexGenerations
	run         RunQuery
	names       PluginNames
	slots       *slots
	maxResults  int
	mu          sync.Mutex
}

// Execute returns the id of the search without waiting for it. A query with no text ends at once with a
// problem; ErrTooManySearches is a start while the most searches kept are all running.
func (u *StartSearch) Execute(ctx context.Context, start searchrun.Start) (searchrun.ID, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	kept := u.dropExpired(1)
	for _, r := range kept {
		if r.IsRunning && searchrun.IsSameSearch(r.Start, start) {
			u.touch(r.ID)
			return r.ID, nil
		}
	}
	if len(kept) >= u.policy.MaxKept {
		return "", searchrun.ErrTooManySearches
	}
	now := u.timer.Now()
	id, err := ulid.NewGenerator(rand.Reader).NewID(now)
	if err != nil {
		return "", err
	}
	record := searchrun.Record{
		ID: searchrun.ID(id), Start: start, IsRunning: true, Step: searchrun.StepQueued,
		IndexGeneration: u.generations.GetGeneration(start.Vault.ID), StartedAt: now, AccessedAt: now,
	}
	if strings.TrimSpace(start.Query) == "" {
		record.IsRunning, record.Step = false, searchrun.StepDone
		record.Errors = []searchrun.Problem{{Message: "the query is empty"}}
		u.store.AddSearch(record)
		return record.ID, nil
	}
	u.store.AddSearch(record)
	runCtx, end := u.cancels.Start(context.WithoutCancel(ctx), id)
	if !start.IsBackground {
		u.tasks.Set(u.newTask(record))
	}
	go u.runSearch(runCtx, end, record)
	return record.ID, nil
}

func (u *StartSearch) newTask(r searchrun.Record) work.Task {
	return work.Task{
		ID: getTaskID(r.ID), Title: taskTitle, About: getAbout(r.Start.Query), IsUserStarted: true,
		PluginID: r.Start.Owner, PluginName: u.names(r.Start.Owner),
		Cancel: func() { u.cancels.Cancel(string(r.ID)) },
	}
}

func getAbout(query string) string {
	about := strings.Join(strings.Fields(query), " ")
	if utf8.RuneCountInString(about) <= taskAboutSize {
		return about
	}
	return string([]rune(about)[:taskAboutSize]) + "…"
}

func (u *StartSearch) runSearch(ctx context.Context, end func(), r searchrun.Record) {
	defer end()
	defer u.tasks.Remove(getTaskID(r.ID))
	if err := u.slots.acquire(ctx, r.Start.Priority); err != nil {
		u.finish(r.ID, searchquery.SearchResults{}, err)
		return
	}
	defer u.slots.release()
	u.store.ChangeSearch(r.ID, func(one *searchrun.Record) { one.Step = searchrun.StepCandidates })
	results, err := u.run(ctx, r.Start.Vault, searchquery.Request{
		Text: r.Start.Query, Limit: u.maxResults, Priority: r.Start.Priority, IndexGeneration: r.IndexGeneration,
		ReportPartial: func(partial searchquery.SearchResults) { u.keepPartial(r.ID, partial) },
	})
	u.finish(r.ID, results, err)
}

func (u *StartSearch) keepPartial(id searchrun.ID, partial searchquery.SearchResults) {
	u.store.ChangeSearch(id, func(r *searchrun.Record) {
		r.Step, r.Results, r.IsPartial = searchrun.StepReranking, partial, true
	})
}

func (u *StartSearch) finish(id searchrun.ID, results searchquery.SearchResults, err error) {
	u.store.ChangeSearch(id, func(r *searchrun.Record) {
		r.IsRunning, r.IsPartial, r.Step = false, false, searchrun.StepDone
		switch {
		case errors.Is(err, context.Canceled):
			r.Results, r.Errors = searchquery.SearchResults{}, []searchrun.Problem{{Message: "the search was canceled"}}
		case err != nil:
			r.Results, r.Errors = searchquery.SearchResults{}, searchrun.ReadProblems(err)
		default:
			r.Results = results
		}
	})
}
