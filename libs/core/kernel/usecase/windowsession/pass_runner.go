package windowsession

import (
	"context"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// failureSink takes what stopped the passes behind a vault.
type failureSink interface {
	SetError(why string)
	SetWatchError(why string)
}

// PassRunner starts the walk, the watch, the indexing and the triggers behind one vault, and hands the
// run back as a value that ends them.
type PassRunner struct {
	failures   failureSink
	diag       *Diagnostics
	closing    <-chan struct{}
	newSession func() IndexSession
	texts      TextVersions
	changes    Changes
	passes     VaultPasses
	triggers   TriggerSource
	wake       IndexWake
}

// NewPassRunner makes the runner over the passes of a vault. closing ends every run when it closes, texts,
// wake and the open of the triggers may be nil.
func NewPassRunner(
	failures *Runtime, diag *Diagnostics, closing <-chan struct{}, newSession func() IndexSession,
	texts TextVersions, changes Changes, passes VaultPasses, triggers TriggerSource, wake IndexWake,
) *PassRunner {
	return &PassRunner{
		failures: failures, diag: diag, closing: closing, newSession: newSession, texts: texts,
		changes: changes, passes: passes, triggers: triggers, wake: wake,
	}
}

func (r *PassRunner) build(ctx context.Context, v domain.Vault, rebuild bool) *runningVault {
	watching, stop := context.WithCancel(context.WithoutCancel(ctx))
	if r.closing != nil {
		go func() {
			select {
			case <-r.closing:
				stop()
			case <-watching.Done():
			}
		}()
	}

	// What a plugin writes as a text version is cut where every other cut happens.
	// A version being made and a vault being scanned are then never two passes
	// over the index at once.
	indexed := r.newSession()
	if r.texts != nil {
		r.texts.SetListener(versionChangeReporter{changes: r.changes, index: indexed})
	}

	// Indexing every file again belongs to the vault this window was opened on,
	// and to nothing built for a vault that arrives later.
	started, ended := r.startIndexing(watching, v, rebuild, indexed)
	if started.Publish != nil {
		started.Publish()
	}
	return &runningVault{retire: started.Retire, cancel: stop, wait: ended}
}

func (r *PassRunner) startIndexing(
	ctx context.Context, v domain.Vault, rebuild bool, indexed IndexSession,
) (Started, func()) {
	// The task is reported before the goroutine starts, so a window opened on a new
	// vault shows the walk from the start.
	r.diag.tasks.AddTask(work.Task{ID: work.WalkingNotes, Title: "Indexing the vault"})

	var running sync.WaitGroup

	triggerPasses := newTriggerPasses()
	started := r.passes.Start(ctx, v, rebuild, Reporters{
		Changed: r.newChangeReporter(triggerPasses, func(paths []string) { r.checkTexts(ctx, v, paths) }),
		Failed:  r.reportFailure,
	})
	if started.Unwatched != nil {
		r.diag.say("not watching %s: %v\n", v.Name, started.Unwatched)
		r.failures.SetWatchError(started.Unwatched.Error())
	}
	running.Go(func() {
		started.Run(ctx)
		// No changes reach the window once the watch stops, so the vault is not
		// followed from this point.
		if started.Unwatched == nil && ctx.Err() == nil {
			r.failures.SetWatchError("the watch stopped")
		}
	})

	running.Go(func() {
		indexed.Run(ctx, v, rebuild, started.Read)
	})

	if runTriggers := r.triggers.start(ctx, v, triggerPasses, func(err error) {
		r.diag.say("run the triggers of %s: %v\n", v.Name, err)
	}); runTriggers != nil {
		running.Go(runTriggers)
	}

	return started, running.Wait
}

func (r *PassRunner) checkTexts(ctx context.Context, v domain.Vault, paths []string) {
	if r.texts == nil {
		return
	}
	if err := r.texts.ReportFilesChanged(ctx, v, paths); err != nil {
		r.diag.say("check the text versions of %s: %v\n", v.Name, err)
	}
}

func (r *PassRunner) reportFailure(err error) {
	if err == nil {
		r.failures.SetError("")
		return
	}
	r.failures.SetError(err.Error())
}

func (r *PassRunner) newChangeReporter(
	triggers *triggerPasses, checkTexts func(paths []string),
) func(paths, assets []string, reload bool) {
	return func(paths, assets []string, reload bool) {
		// A client shows every file in the vault, so an asset is reported as a note is.
		changePaths := slices.Concat(paths, assets)
		if reload {
			triggers.requestAll()
		} else {
			triggers.requestPaths(changePaths)
		}
		r.changes.Publish(window.Change{Paths: changePaths, ShouldReload: reload})
		checkTexts(changePaths)
		if r.wake == nil {
			return
		}
		r.wake.ClearSkips(changePaths)
		if reload || len(assets) > 0 {
			// A book added to an open vault is read without a request.
			r.wake.WakeSources()
		}
		if len(paths) > 0 {
			// A written note is chunked again, and its chunks need new vectors. The index
			// determines which.
			r.wake.WakeNotes()
		}
	}
}

// versionChangeReporter asks the index session to cut a file whose text versions changed, and
// reports the file to the window.
type versionChangeReporter struct {
	changes Changes
	index   IndexSession
}

func (r versionChangeReporter) ReportVersionsChanged(v domain.Vault, path string) {
	r.index.ReportVersionsChanged(v, path)
	r.changes.Publish(window.Change{VersionPaths: []string{path}})
}
