// Package teardown stops the processes of the plugins.
package teardown

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
)

// eventCloser stops the delivery of the events of the application.
type eventCloser interface {
	CloseBus(ctx context.Context) error
}

// listenerControl syncs the ports bound for the listeners of the plugins with the host, and closes them.
type listenerControl interface {
	SyncAll(ctx context.Context)
	CloseAll()
}

// evictionTimer ends the timer that stops the idle plugins.
type evictionTimer interface {
	Stop()
	IsRunning() bool
}

// modeState is what the teardown sets of the mode of the host.
type modeState interface {
	SetSafe(enabled bool)
	Close()
}

// Teardown stops the processes of the plugins: one on request, the idle ones on a timer, all of them
// when the host enters safe mode, and all of them when the host shuts down.
type Teardown struct {
	mode      modeState
	sups      *supervision.Supervisors
	stopper   *supervision.Stopper
	evictor   evictionTimer
	events    eventCloser
	listeners listenerControl
	report    func(error)
}

// Deps are what a teardown stops and sets: the mode of the host, the supervisors, the stopper, the timer of the evictions, the
// events and the listeners. Report receives the errors of a stop.
type Deps struct {
	Mode      modeState
	Sups      *supervision.Supervisors
	Stopper   *supervision.Stopper
	Evictor   evictionTimer
	Events    eventCloser
	Listeners listenerControl
	Report    func(error)
}

// New returns the teardown over deps.
func New(deps Deps) *Teardown {
	return &Teardown{
		mode: deps.Mode, sups: deps.Sups, stopper: deps.Stopper, evictor: deps.Evictor, events: deps.Events,
		listeners: deps.Listeners, report: deps.Report,
	}
}

// StopPlugin stops the process of the plugin, if one runs, and forgets its endpoint.
func (t *Teardown) StopPlugin(ctx context.Context, pluginID string) error {
	return t.stopper.StopPlugin(ctx, pluginID)
}

// EvictIdle stops the plugins that hold no lease and have been idle for longer than threshold.
func (t *Teardown) EvictIdle(ctx context.Context, threshold, gracePeriod time.Duration) []string {
	return t.stopper.EvictIdle(ctx, threshold, gracePeriod)
}

// SetSafeMode enables or disables safe recovery mode. Entering it stops every running plugin.
func (t *Teardown) SetSafeMode(enabled bool) {
	t.mode.SetSafe(enabled)
	if enabled {
		for _, id := range t.sups.ListRunning() {
			if err := t.stopper.StopPlugin(context.Background(), id); err != nil {
				t.report(err)
			}
		}
	}
	t.listeners.SyncAll(context.Background())
}

// TeardownAll shuts down all active plugin processes.
func (t *Teardown) TeardownAll(ctx context.Context, gracePeriod time.Duration) error {
	t.evictor.Stop()

	firstErr := t.events.CloseBus(ctx)

	t.mode.Close()
	targets := t.stopper.TakeRunning()
	t.listeners.CloseAll()

	if err := t.stopper.StopTargets(ctx, targets, gracePeriod); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// IsEvicting reports whether the idle plugins are stopped on a timer.
func (t *Teardown) IsEvicting() bool {
	return t.evictor.IsRunning()
}

// EvictTargets stops the targets and returns the ids of the ones it stopped.
func (t *Teardown) EvictTargets(ctx context.Context, targets []supervision.Target, gracePeriod time.Duration) []string {
	return t.stopper.EvictTargets(ctx, targets, gracePeriod)
}
