package windowsession

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// lifecycle starts and stops the passes behind the vault of a window. Every scenario that changes
// what runs behind the vault goes through it.
type lifecycle struct {
	rt   *Runtime
	deps Deps
}

func (l *lifecycle) say(format string, a ...any) {
	if _, err := fmt.Fprintf(l.deps.Out, format, a...); err != nil {
		l.deps.Tasks.AddTask(work.Task{ID: writingDiagnostics, Title: "Writing diagnostics", Error: err.Error()})
	}
}

// writingDiagnostics is the entry in the list of what is being done for a diagnostic line that could
// not be written.
const writingDiagnostics = "writing diagnostics"

func (l *lifecycle) startVault(ctx context.Context, v domain.Vault, rebuild bool) {
	l.rt.SetVault(v)
	if v.ID == "" {
		// Nothing is being read, so nothing is waited for.
		l.rt.SetReady(true)
		return
	}
	// Recorded before the vault is built, so the next window opens on it. A failure
	// to write the list is reported and does not stop the vault from opening.
	if err := l.deps.Recorded.Execute(v.ID); err != nil {
		l.say("not recording %s as the vault opened: %v\n", v.Name, err)
	}
	// Published last, so a request that reads a run reads the one belonging to the vault in front of it.
	l.rt.running.Store(l.buildSession(ctx, v, rebuild))
}

func (l *lifecycle) buildSession(ctx context.Context, v domain.Vault, rebuild bool) *runningVault {
	watching, stop := context.WithCancel(context.WithoutCancel(ctx))
	if l.deps.Closing != nil {
		go func() {
			select {
			case <-l.deps.Closing:
				stop()
			case <-watching.Done():
			}
		}()
	}

	// What a plugin writes as a text version is cut where every other cut happens.
	// A version being made and a vault being scanned are then never two passes
	// over the index at once.
	indexed := l.deps.NewSession()
	if l.deps.Texts != nil {
		l.deps.Texts.SetListener(versionChangeReporter{changes: l.deps.Changes, index: indexed})
	}

	// Indexing every file again belongs to the vault this window was opened on,
	// and to nothing built for a vault that arrives later.
	started, ended := l.startIndexing(watching, v, rebuild, indexed)
	if started.Publish != nil {
		started.Publish()
	}
	return &runningVault{retire: started.Retire, cancel: stop, wait: ended}
}

func (l *lifecycle) startIndexing(
	ctx context.Context, v domain.Vault, rebuild bool, indexed IndexSession,
) (Started, func()) {
	// The task is reported before the goroutine starts, so a window opened on a new
	// vault shows the walk from the start.
	l.deps.Tasks.AddTask(work.Task{ID: work.WalkingNotes, Title: "Indexing the vault"})

	var running sync.WaitGroup

	triggerPasses := newTriggerPasses()
	started := l.deps.Passes.Start(ctx, v, rebuild, Reporters{
		Changed: l.newChangeReporter(triggerPasses, func(paths []string) { l.checkTexts(ctx, v, paths) }),
		Failed:  l.reportFailure,
	})
	if started.Unwatched != nil {
		l.say("not watching %s: %v\n", v.Name, started.Unwatched)
		l.rt.SetWatchError(started.Unwatched.Error())
	}
	running.Go(func() {
		started.Run(ctx)
		// No changes reach the window once the watch stops, so the vault is not
		// followed from this point.
		if started.Unwatched == nil && ctx.Err() == nil {
			l.rt.SetWatchError("the watch stopped")
		}
	})

	running.Go(func() {
		indexed.Run(ctx, v, rebuild, started.Read)
	})

	if runTriggers := l.deps.Triggers.start(ctx, v, triggerPasses, func(err error) {
		l.say("run the triggers of %s: %v\n", v.Name, err)
	}); runTriggers != nil {
		running.Go(runTriggers)
	}

	return started, running.Wait
}

func (l *lifecycle) checkTexts(ctx context.Context, v domain.Vault, paths []string) {
	if l.deps.Texts == nil {
		return
	}
	if err := l.deps.Texts.ReportFilesChanged(ctx, v, paths); err != nil {
		l.say("check the text versions of %s: %v\n", v.Name, err)
	}
}

func (l *lifecycle) reportFailure(err error) {
	if err == nil {
		l.rt.SetError("")
		return
	}
	l.rt.SetError(err.Error())
}

func (l *lifecycle) newChangeReporter(
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
		l.deps.Changes.Publish(window.Change{Paths: changePaths, ShouldReload: reload})
		checkTexts(changePaths)
		if l.deps.Wake == nil {
			return
		}
		l.deps.Wake.ClearSkips(changePaths)
		if reload || len(assets) > 0 {
			// A book added to an open vault is read without a request.
			l.deps.Wake.WakeSources()
		}
		if len(paths) > 0 {
			// A written note is chunked again, and its chunks need new vectors. The index
			// determines which.
			l.deps.Wake.WakeNotes()
		}
	}
}

func (l *lifecycle) stopVault() {
	on := l.rt.running.Swap(nil)
	if on == nil {
		return
	}
	if on.retire != nil {
		on.retire()
	}
	on.cancel()
	on.wait()
}

func (l *lifecycle) resetStatus() {
	l.rt.SetReady(false)
	l.rt.SetError("")
	l.rt.SetWatchError("")

	for _, pass := range []string{work.WalkingNotes, work.IndexingTexts, work.MakingVectors, work.WordsAlone} {
		l.deps.Tasks.RemoveTask(pass)
	}
	if l.deps.EmbedderError != nil {
		l.deps.Tasks.AddTask(work.Task{ID: work.MakingVectors, Title: "Indexing", Error: l.deps.EmbedderError.Error()})
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
