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
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/resultcache"
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
// another in path order, every saved search that is due: one without a cache, one whose query changed
// since its run, and one marked stale whose cooldown since its last run ended or, while a tab of it is
// open, whose short debounce since it went stale ended. Its searches are background searches, for which
// the core shows no task; it reports the runs as one task of the plugin. When a stale search waits, a
// timer runs RunDue again at the end of its wait, for as long as the process lives.
func (r *Runner) RunDue(ctx context.Context, vault domain.VaultID) error {
	unlock, err := r.vaults.lock(ctx, vault)
	if err != nil {
		return err
	}
	defer unlock()
	v := r.open(vault)
	searches, err := listSearches(ctx, v)
	if err != nil {
		return err
	}
	if err := deleteOtherFiles(ctx, v, searches); err != nil {
		return err
	}
	settings := r.settings()
	due, wakeAt := r.listDueSearches(ctx, vault, v, searches, settings)
	r.schedule(ctx, vault, wakeAt)
	if len(due) == 0 {
		return nil
	}
	return r.runBatch(ctx, job{vaultID: vault, vault: v, path: "", priority: host.SearchPriorityBackground, isBackground: true}, due)
}

// RunNow runs one saved search at once, whatever its marks, after the runs of the vault on their way. Its
// search is interactive and the core shows it as a task. The note is running from the call, the wait for
// the runs on their way included.
func (r *Runner) RunNow(ctx context.Context, vault domain.VaultID, notePath string) (resultcache.Record, error) {
	r.watchers.begin(watchKey{vault: vault, path: notePath})
	return r.runNowBegun(ctx, vault, notePath)
}

func (r *Runner) runNowBegun(ctx context.Context, vault domain.VaultID, notePath string) (resultcache.Record, error) {
	unlock, err := r.vaults.lock(ctx, vault)
	if err != nil {
		r.watchers.end(watchKey{vault: vault, path: notePath}, time.Time{})
		return resultcache.Record{}, err
	}
	defer unlock()
	return r.runBegun(ctx, job{
		vaultID: vault, vault: r.open(vault), path: notePath, priority: host.SearchPriorityInteractive, isBackground: false,
	})
}

func (r *Runner) listDueSearches(ctx context.Context, vault domain.VaultID, v Vault, searches []search, settings Settings) ([]search, time.Time) {
	now := r.now()
	var due []search
	var wakeAt time.Time
	for _, one := range searches {
		raw, stat, err := v.Notes.Read(ctx, one.path)
		if err != nil {
			due = append(due, one)
			continue
		}
		record, hasRecord, err := readRecord(ctx, v.Store, one.name)
		switch {
		case err != nil, !hasRecord, isChanged(readTarget(stat, raw), record):
			due = append(due, one)
			continue
		case record.StaleSince == nil:
			continue
		}
		runAt := getRunTime(record, settings, r.watchers.isWatched(watchKey{vault: vault, path: one.path}))
		switch {
		case !now.Before(runAt):
			due = append(due, one)
		case wakeAt.IsZero() || runAt.Before(wakeAt):
			wakeAt = runAt
		}
	}
	return due, wakeAt
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

func (r *Runner) runBatch(ctx context.Context, batch job, due []search) (err error) {
	task := host.Task{
		ID: taskID, Title: taskTitle, Detail: "", CompletedCount: 0, TotalCount: int64(len(due)), Unit: host.UnitThings,
		FailureReason: "", IsUserStarted: false,
	}
	reportErr := r.tasks.ReportTask(ctx, task)
	defer func() {
		err = errors.Join(err, r.tasks.FinishTask(context.WithoutCancel(ctx), taskID))
	}()
	var failures []error
	var reasons []string
	for i, one := range due {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		batch.path = one.path
		if _, runErr := r.run(ctx, batch); runErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", one.path, runErr))
			reasons = append(reasons, fmt.Sprintf("%s: %v", one.path, runErr))
		}
		task.CompletedCount, task.FailureReason = int64(i+1), strings.Join(reasons, "; ")
		reportErr = errors.Join(reportErr, r.tasks.ReportTask(ctx, task))
	}
	return errors.Join(errors.Join(failures...), reportErr)
}

func listSearches(ctx context.Context, v Vault) ([]search, error) {
	paths, err := v.Notes.GetPathsOfType(ctx, searchnote.NoteType)
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

// StartRunDue runs RunDue in the background until ctx ends, as the opening of a tab and a change of the
// settings do.
func (r *Runner) StartRunDue(ctx context.Context, vault domain.VaultID) {
	go func() {
		if err := r.RunDue(ctx, vault); err != nil {
			r.log.Debug("saved search: run the due searches", "vault", vault, "error", err)
		}
	}()
}

// StartRunNow runs RunNow in the background until ctx ends, as an edit of the search in the tab does. It
// cancels the check or run of the previous edit of the note, and the next edit of the note or the end of
// its last watch cancels it. The note is running from the call to the end of the run, the wait for the
// runs of the vault on their way included.
func (r *Runner) StartRunNow(ctx context.Context, vault domain.VaultID, notePath string) {
	key := watchKey{vault: vault, path: notePath}
	ctx, end := r.edits.start(ctx, key, true)
	r.watchers.begin(key)
	go func() {
		defer end()
		if _, err := r.runNowBegun(ctx, vault, notePath); err != nil {
			r.log.Debug("saved search: run an edited search", "vault", vault, "path", notePath, "error", err)
		}
	}()
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
