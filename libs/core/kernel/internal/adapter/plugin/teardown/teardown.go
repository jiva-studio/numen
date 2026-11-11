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

// New returns the teardown over the mode of the host, the supervisors, the stopper, the timer of the evictions, the events and the listeners; report receives
// the errors of a stop.
func New(
	mode modeState, sups *supervision.Supervisors, stopper *supervision.Stopper, evictor evictionTimer, events eventCloser,
	listeners listenerControl, report func(error),
) *Teardown {
	return &Teardown{mode: mode, sups: sups, stopper: stopper, evictor: evictor, events: events, listeners: listeners, report: report}
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
