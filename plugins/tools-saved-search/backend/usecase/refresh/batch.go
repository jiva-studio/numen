package refresh

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/domain/resultcache"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/searchnote"
)

const (
	taskID    = "saved-search-refresh"
	taskTitle = "Refreshing saved searches"
)

// ErrNotASearch is what a note gets that is not of type search.
var ErrNotASearch = searchnote.ErrNotASearch

// search is a saved search note of a vault and the name of its cache file.
type search struct {
	path string
	name string
}

// MarkStale marks every saved search of the vault stale from now, where it is not stale already. A search
// without a cache gets none: it is due anyway. It does not wait for a run of the vault.
func (r *Runner) MarkStale(ctx context.Context, vault domain.VaultID) error {
	v := r.open(vault)
	searches, err := listSearches(ctx, v)
	if err != nil {
		return err
	}
	now := r.now()
	r.records.Lock()
	defer r.records.Unlock()
	for _, one := range searches {
		record, hasRecord, err := readRecord(ctx, v.Store, one.name)
		if err != nil {
			return err
		}
		if !hasRecord || record.StaleSince != nil {
			continue
		}
		record.StaleSince = &now
		if err := writeRecord(ctx, v.Store, one.name, record); err != nil {
			return err
		}
	}
	return nil
}

// RunDue deletes the caches of notes that are no longer saved searches of the vault and runs, one after
// another, every saved search that is due: one without a cache, one whose query changed since its run,
// and one marked stale whose cooldown since its last run ended or, while a tab of it is open, whose short
// debounce since it went stale ended. The searches open in a tab go first, the others in path order. A
// search without a good run is running from the start of the batch. Before each search the batch lets the
// interactive runs of the vault that wait go first, and an interactive run that comes cancels the search
// of the batch on its way, which the batch runs again after it. Its searches are background searches, for
// which the core shows no task; it reports the runs as one task of the plugin. When a stale search waits,
// a timer runs RunDue again at the end of its wait, for as long as the process lives.
func (r *Runner) RunDue(ctx context.Context, vault domain.VaultID) error {
	unlock, err := r.vaults.lockBatch(ctx, vault)
	if err != nil {
		return err
	}
	defer unlock()
	v := r.open(vault)
	due, err := r.listDueSearches(ctx, vault, v)
	if err != nil || len(due) == 0 {
		return err
	}
	return r.runBatch(ctx, vault, v, due)
}

// RunNow runs one saved search at once, whatever its marks, before the searches of a batch of the vault
// that wait, and after the interactive runs of the vault on their way. Its search is interactive and the
// core shows it as a task. The note is running from the call, the wait for the runs on their way included.
func (r *Runner) RunNow(ctx context.Context, vault domain.VaultID, notePath string) (resultcache.Record, error) {
	r.watchers.begin(watchKey{vault: vault, path: notePath})
	return r.runNowBegun(ctx, vault, notePath)
}

func (r *Runner) runNowBegun(ctx context.Context, vault domain.VaultID, notePath string) (resultcache.Record, error) {
	unlock, err := r.vaults.lockInteractive(ctx, vault)
	if err != nil {
		r.watchers.end(watchKey{vault: vault, path: notePath}, time.Time{})
		return resultcache.Record{}, err
	}
	defer unlock()
	return r.runBegun(ctx, r.newInteractiveJob(vault, notePath))
}

func (r *Runner) newInteractiveJob(vault domain.VaultID, notePath string) job {
	return job{vaultID: vault, vault: r.open(vault), path: notePath, priority: host.SearchPriorityInteractive, isBackground: false}
}

func (r *Runner) runIfDue(ctx context.Context, vault domain.VaultID, notePath string) error {
	unlock, err := r.vaults.lockInteractive(ctx, vault)
	if err != nil {
		return err
	}
	defer unlock()
	one := r.newInteractiveJob(vault, notePath)
	check := r.checkDueSearch(ctx, vault, one.vault, search{path: notePath, name: r.findFileName(ctx, one)}, r.getSettings(), r.now())
	if !check.isDue {
		return nil
	}
	_, err = r.run(ctx, one)
	return err
}

// dueSearch is a saved search a batch runs, and whether it has no good run yet.
type dueSearch struct {
	search
	isNew bool
}

// dueCheck is whether a saved search is due, the time it is due at when it is not, whether it has no good
// run yet and whether a tab of it is open.
type dueCheck struct {
	isDue     bool
	isNew     bool
	isWatched bool
	runAt     time.Time
}

func (r *Runner) listDueSearches(ctx context.Context, vault domain.VaultID, v Vault) ([]dueSearch, error) {
	_, unlock, err := r.vaults.lockBackground(ctx, vault)
	if err != nil {
		return nil, err
	}
	defer unlock()
	searches, err := listSearches(ctx, v)
	if err != nil {
		return nil, err
	}
	if err := deleteOtherFiles(ctx, v, searches); err != nil {
		return nil, err
	}
	settings, now := r.getSettings(), r.now()
	var due, watched []dueSearch
	var wakeAt time.Time
	for _, one := range searches {
		check := r.checkDueSearch(ctx, vault, v, one, settings, now)
		switch {
		case check.isDue && check.isWatched:
			watched = append(watched, dueSearch{search: one, isNew: check.isNew})
		case check.isDue:
			due = append(due, dueSearch{search: one, isNew: check.isNew})
		case !check.runAt.IsZero() && (wakeAt.IsZero() || check.runAt.Before(wakeAt)):
			wakeAt = check.runAt
		}
	}
	r.schedule(ctx, vault, wakeAt)
	return append(watched, due...), nil
}

func (r *Runner) checkDueSearch(ctx context.Context, vault domain.VaultID, v Vault, one search, settings Settings, now time.Time) dueCheck {
	isWatched := r.watchers.isWatched(watchKey{vault: vault, path: one.path})
	check := dueCheck{isDue: true, isNew: false, isWatched: isWatched, runAt: time.Time{}}
	raw, stat, err := v.Notes.Read(ctx, one.path)
	if err != nil {
		return check
	}
	record, hasRecord, err := readRecord(ctx, v.Store, one.name)
	switch {
	case err != nil:
		return check
	case !hasRecord:
		check.isNew = true
		return check
	}
	check.isNew = record.ComputedAt.IsZero()
	switch {
	case isChanged(readTarget(stat, raw), record):
		return check
	case record.StaleSince == nil:
		check.isDue = false
		return check
	}
	runAt := getRunTime(record, settings, isWatched)
	if now.Before(runAt) {
		check.isDue, check.runAt = false, runAt
	}
	return check
}

func (r *Runner) schedule(ctx context.Context, vault domain.VaultID, wakeAt time.Time) {
	var delay time.Duration
	if !wakeAt.IsZero() {
		delay = max(wakeAt.Sub(r.clock.Now()), time.Second)
	}
	detached := context.WithoutCancel(ctx)
	r.timers.set(vault, delay, func() {
		if err := r.RunDue(detached, vault); err != nil {
			r.log.Debug("saved search: run the due searches at the end of the cooldown", "vault", vault, "error", err)
		}
	})
}

func (r *Runner) runBatch(ctx context.Context, vault domain.VaultID, v Vault, due []dueSearch) (err error) {
	task := host.Task{
		ID: taskID, Title: taskTitle, Detail: "", CompletedCount: 0, TotalCount: int64(len(due)), Unit: host.UnitThings,
		FailureReason: "", IsUserStarted: false,
	}
	reportErr := r.tasks.ReportTask(ctx, task)
	defer func() {
		err = errors.Join(err, r.tasks.FinishTask(context.WithoutCancel(ctx), taskID))
	}()
	for _, one := range due {
		if one.isNew {
			r.watchers.queue(watchKey{vault: vault, path: one.path})
		}
	}
	next := 0
	defer func() {
		for _, one := range due[next:] {
			if one.isNew {
				r.watchers.dequeue(watchKey{vault: vault, path: one.path})
			}
		}
	}()
	var failures []error
	var reasons []string
	for next < len(due) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		one := due[next]
		isYielded, runErr := r.runQueuedSearch(ctx, vault, v, one)
		if isYielded {
			continue
		}
		next++
		if runErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", one.path, runErr))
			reasons = append(reasons, fmt.Sprintf("%s: %v", one.path, runErr))
		}
		task.CompletedCount, task.FailureReason = int64(next), strings.Join(reasons, "; ")
		reportErr = errors.Join(reportErr, r.tasks.ReportTask(ctx, task))
	}
	return errors.Join(errors.Join(failures...), reportErr)
}

func (r *Runner) runQueuedSearch(ctx context.Context, vault domain.VaultID, v Vault, one dueSearch) (bool, error) {
	key := watchKey{vault: vault, path: one.path}
	runCtx, unlock, err := r.vaults.lockBackground(ctx, vault)
	if err != nil {
		r.watchers.dequeue(key)
		return false, err
	}
	defer unlock()
	if !r.checkDueSearch(runCtx, vault, v, one.search, r.getSettings(), r.now()).isDue {
		r.watchers.dequeue(key)
		return false, nil
	}
	batch := job{vaultID: vault, vault: v, path: one.path, priority: host.SearchPriorityBackground, isBackground: true}
	_, err = r.run(runCtx, batch)
	if ctx.Err() == nil && errors.Is(context.Cause(runCtx), errYielded) {
		return true, nil
	}
	r.watchers.dequeue(key)
	return false, err
}

func listSearches(ctx context.Context, v Vault) ([]search, error) {
	paths, err := v.Notes.ListPathsOfType(ctx, searchnote.NoteType)
	if err != nil {
		return nil, err
	}
	slices.Sort(paths)
	refs, err := v.Lookup.GetNotes(ctx, paths)
	if err != nil {
		return nil, err
	}
	searches := make([]search, len(paths))
	for i, notePath := range paths {
		searches[i] = search{path: notePath, name: resultcache.GetFileName(refs[notePath].ID, notePath)}
	}
	return searches, nil
}

func deleteOtherFiles(ctx context.Context, v Vault, searches []search) error {
	files, err := v.Store.List(ctx)
	if err != nil {
		return err
	}
	names := make(map[string]bool, len(searches))
	for _, one := range searches {
		names[one.name] = true
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name, ".json") || names[file.Name] {
			continue
		}
		if err := v.Store.Delete(ctx, file.Name); err != nil {
			return err
		}
	}
	return nil
}

// StartRunDue runs RunDue in the background until ctx ends, as an index change, the opening of a tab and a
// change of the settings do.
func (r *Runner) StartRunDue(ctx context.Context, vault domain.VaultID) {
	r.started.Go(func() {
		if err := r.RunDue(ctx, vault); err != nil {
			r.log.Debug("saved search: run the due searches", "vault", vault, "error", err)
		}
	})
}

// StartRunIfDue runs the saved search of the note in the background until ctx ends, as an interactive run
// before the searches of a batch of the vault that wait, when it is due as RunDue reads it with a tab of
// it open; the opening of its tab does so.
func (r *Runner) StartRunIfDue(ctx context.Context, vault domain.VaultID, notePath string) {
	r.started.Go(func() {
		if err := r.runIfDue(ctx, vault, notePath); err != nil {
			r.log.Debug("saved search: run an opened search", "vault", vault, "path", notePath, "error", err)
		}
	})
}

// StartRunNow runs RunNow in the background until ctx ends, as an edit of the search in the tab does. It
// cancels the check or run of the previous edit of the note, and the next edit of the note or the end of
// its last watch cancels it. The note is running from the call to the end of the run, the wait for the
// runs of the vault on their way included.
func (r *Runner) StartRunNow(ctx context.Context, vault domain.VaultID, notePath string) {
	key := watchKey{vault: vault, path: notePath}
	ctx, end := r.edits.start(ctx, key, true)
	r.watchers.begin(key)
	r.started.Go(func() {
		defer end()
		if _, err := r.runNowBegun(ctx, vault, notePath); err != nil {
			r.log.Debug("saved search: run an edited search", "vault", vault, "path", notePath, "error", err)
		}
	})
}

// CheckQuery runs the query on the core as an interactive search and returns its problems, counted from
// the first line of the query; none when the query is valid and its providers answered.
func (r *Runner) CheckQuery(ctx context.Context, vault domain.VaultID, query string) ([]resultcache.Problem, error) {
	start := host.SearchStart{Query: query, Priority: host.SearchPriorityInteractive, IsBackground: false}
	result, err := r.runSearch(ctx, r.open(vault).Search, start)
	if err != nil {
		return nil, err
	}
	return convertSearchProblems(result.state.Errors), nil
}

// CheckNoteQuery is CheckQuery for an edit of the note at notePath, and the note is running while it
// checks. It cancels the check or run of the previous edit of the note; the next edit of the note cancels
// it, and it then returns context.Canceled.
func (r *Runner) CheckNoteQuery(ctx context.Context, vault domain.VaultID, notePath string, query string) ([]resultcache.Problem, error) {
	key := watchKey{vault: vault, path: notePath}
	ctx, end := r.edits.start(ctx, key, false)
	defer end()
	r.watchers.begin(key)
	defer r.watchers.end(key, time.Time{})
	return r.CheckQuery(ctx, vault, query)
}
