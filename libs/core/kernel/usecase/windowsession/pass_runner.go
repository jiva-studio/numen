package windowsession

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// FailureSink takes what stopped the passes behind a vault.
type FailureSink interface {
	SetError(why string)
	SetWatchError(why string)
}

// PassRunner starts the walk, the watch, the indexing and the triggers behind one vault, and hands the
// run back as a value that ends them.
type PassRunner struct {
	failures   FailureSink
	diag       *Diagnostics
	closing    <-chan struct{}
	newSession func() IndexSession
	reporter   ChangeReporting
	passes     VaultPasses
	triggers   TriggerSource
}

// PassRunnerConfig is what a PassRunner is made of. Closing ends every run when it closes. The open of
// Triggers may be nil.
type PassRunnerConfig struct {
	Failures   FailureSink
	Diag       *Diagnostics
	Closing    <-chan struct{}
	NewSession func() IndexSession
	Reporter   ChangeReporting
	Passes     VaultPasses
	Triggers   TriggerSource
}

// NewPassRunner makes the runner over the passes of a vault.
func NewPassRunner(cfg PassRunnerConfig) *PassRunner {
	return &PassRunner{
		failures: cfg.Failures, diag: cfg.Diag, closing: cfg.Closing, newSession: cfg.NewSession,
		reporter: cfg.Reporter, passes: cfg.Passes, triggers: cfg.Triggers,
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
	triggerPasses := newTriggerPasses()
	r.reporter.listenVersions(indexed, triggerPasses)

	// Indexing every file again belongs to the vault this window was opened on,
	// and to nothing built for a vault that arrives later.
	started, ended := r.startIndexing(watching, v, rebuild, indexed, triggerPasses)
	if started.Publish != nil {
		started.Publish()
	}
	return &runningVault{retire: started.Retire, cancel: stop, wait: ended}
}

func (r *PassRunner) startIndexing(
	ctx context.Context, v domain.Vault, rebuild bool, indexed IndexSession, triggerPasses *triggerPasses,
) (Started, func()) {
	// The task is reported before the goroutine starts, so a window opened on a new
	// vault shows the walk from the start.
	r.diag.tasks.AddTask(work.Task{ID: work.WalkingNotes, Title: "Indexing the vault"})

	var running sync.WaitGroup

	started := r.passes.Start(ctx, v, rebuild, Reporters{
		Changed: r.reporter.newChangeReporter(triggerPasses, func(paths []string) { r.reporter.checkTexts(ctx, v, paths) }),
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

func (r *PassRunner) reportFailure(err error) {
	if err == nil {
		r.failures.SetError("")
		return
	}
	r.failures.SetError(err.Error())
}

// ChangeReporting tells the window, the index and the text versions what the passes behind a vault
// found changed. ChangeReporter is its implementation.
type ChangeReporting interface {
	listenVersions(indexed IndexSession, triggers *triggerPasses)
	checkTexts(ctx context.Context, v domain.Vault, paths []string)
	newChangeReporter(triggers *triggerPasses, checkTexts func(paths []string)) func(paths, assets []string, reload bool)
}
