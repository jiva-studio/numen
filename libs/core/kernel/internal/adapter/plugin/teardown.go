package plugin

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
)

// pluginStopper stops the process of a plugin.
type pluginStopper interface {
	StopPlugin(ctx context.Context, pluginID string) error
}

// eventCloser stops the delivery of the events of the application.
type eventCloser interface {
	closeBus(ctx context.Context) error
}

// listenerControl syncs the ports bound for the listeners of the plugins with the host, and closes them.
type listenerControl interface {
	syncAll(ctx context.Context)
	closeAll()
}

// Teardown stops the processes of the plugins: one on request, the idle ones on a timer, all of them
// when the host enters safe mode, and all of them when the host shuts down.
type Teardown struct {
	mode      *Mode
	sups      *supervision.Supervisors
	stopper   *supervision.Stopper
	events    eventCloser
	listeners listenerControl
	report    func(error)
}

func newTeardown(
	mode *Mode, sups *supervision.Supervisors, stopper *supervision.Stopper, events eventCloser,
	listeners listenerControl, report func(error),
) *Teardown {
	return &Teardown{mode: mode, sups: sups, stopper: stopper, events: events, listeners: listeners, report: report}
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
	t.mode.setSafe(enabled)
	if enabled {
		for _, id := range t.sups.ListRunning() {
			if err := t.stopper.StopPlugin(context.Background(), id); err != nil {
				t.report(err)
			}
		}
	}
	t.listeners.syncAll(context.Background())
}

// TeardownAll shuts down all active plugin processes.
func (t *Teardown) TeardownAll(ctx context.Context, gracePeriod time.Duration) error {
	t.stopper.StopEvicting()

	firstErr := t.events.closeBus(ctx)

	t.mode.close()
	targets := t.stopper.TakeRunning()
	t.listeners.closeAll()

	if err := t.stopper.StopTargets(ctx, targets, gracePeriod); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}
