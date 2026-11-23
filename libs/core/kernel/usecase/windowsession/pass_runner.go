package windowsession

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
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
	newSession func(domain.Vault) IndexSession
	reporter   ChangeReporting
	passes     VaultPasses
	triggers   TriggerSource
	startFeed  func(ctx context.Context, v domain.Vault) func()
	report     func(domain.Vault, string, error)
}

// PassRunnerConfig is what a PassRunner is made of. Closing ends every run when it closes. Triggers.Open and
// StartFeed may be nil.
type PassRunnerConfig struct {
	Failures   FailureSink
	Diag       *Diagnostics
	Closing    <-chan struct{}
	NewSession func(domain.Vault) IndexSession
	Reporter   ChangeReporting
	Passes     VaultPasses
	Triggers   TriggerSource
	StartFeed  func(ctx context.Context, v domain.Vault) func()
	Report     func(domain.Vault, string, error)
}

// NewPassRunner makes the runner over the passes of a vault.
func NewPassRunner(cfg PassRunnerConfig) *PassRunner {
	return &PassRunner{
		failures: cfg.Failures, diag: cfg.Diag, closing: cfg.Closing, newSession: cfg.NewSession,
		reporter: cfg.Reporter, passes: cfg.Passes, triggers: cfg.Triggers, startFeed: cfg.StartFeed, report: cfg.Report,
	}
}

func (r *PassRunner) openIndexSession(v domain.Vault) IndexSession { return r.newSession(v) }

func (r *PassRunner) build(ctx context.Context, v domain.Vault, rebuild bool, indexed IndexSession) *runningVault {
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
	r.reporter.listenVersions(indexed)

	// Indexing every file again belongs to the vault this window was opened on,
	// and to nothing built for a vault that arrives later.
	started, ended := r.startIndexing(watching, v, rebuild, indexed)
	if started.Publish != nil {
		started.Publish()
	}
	r.reportEvent(v, "Vault opened", nil)
	return &runningVault{retire: started.Retire, cancel: stop, wait: func() {
		ended()
		r.reportEvent(v, "Vault closed", nil)
	}}
}

func (r *PassRunner) startIndexing(
	ctx context.Context, v domain.Vault, rebuild bool, indexed IndexSession,
) (Started, func()) {
	// The task is reported before the goroutine starts, so a window opened on a new
	// vault shows the walk from the start.
	r.diag.tasks.AddTask(work.Task{ID: work.WalkingNotes, Title: "Indexing the vault"})

	var running sync.WaitGroup

	var runFeed func()
	if r.startFeed != nil {
		runFeed = r.startFeed(ctx, v)
	}
	triggerPasses := newTriggerPasses()
	started := r.passes.Start(ctx, v, rebuild, Reporters{
		NotifyChanged: r.reporter.newChangeReporter(triggerPasses, func(paths []string) { r.reporter.checkTexts(ctx, v, paths) }),
		Failed:        r.reportFailure,
	})
	if started.Unwatched != nil {
		r.diag.say("not watching %s: %v\n", v.Name, started.Unwatched)
		r.failures.SetWatchError(started.Unwatched.Error())
		r.reportEvent(v, "Vault watch unavailable", started.Unwatched)
	}
	running.Go(func() {
		started.Run(ctx)
		// No changes reach the window once the watch stops, so the vault is not
		// followed from this point.
		if started.Unwatched == nil && ctx.Err() == nil {
			r.failures.SetWatchError("the watch stopped")
			r.reportEvent(v, "Vault watch stopped", nil)
		}
	})

	running.Go(func() {
		indexed.Run(ctx, rebuild, started.Read)
	})

	if runTriggers := r.triggers.start(ctx, v, triggerPasses, func(err error) {
		r.diag.say("run the triggers of %s: %v\n", v.Name, err)
		r.reportEvent(v, "Vault trigger failed", err)
	}); runTriggers != nil {
		running.Go(runTriggers)
	}

	if runFeed != nil {
		running.Go(runFeed)
	}

	return started, running.Wait
}

func (r *PassRunner) reportEvent(v domain.Vault, event string, err error) {
	if r.report != nil {
		r.report(v, event, err)
	}
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
	listenVersions(indexed IndexSession)
	checkTexts(ctx context.Context, v domain.Vault, paths []string)
	newChangeReporter(triggers *triggerPasses, checkTexts func(paths []string)) func(paths, assets []string, reload bool)
}
