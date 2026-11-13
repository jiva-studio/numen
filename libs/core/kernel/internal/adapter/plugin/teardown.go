package plugin

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
)

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

// healthRemover takes health items of the plugins out of their list.
type healthRemover interface {
	RemovePlugin(pluginID string)
	RemoveAll()
}

// evictionTimer ends the timer that stops the idle plugins.
type evictionTimer interface {
	Stop()
	IsRunning() bool
}

// Teardown stops the processes of the plugins: one on request, the idle ones on a timer, all of them
// when the host enters safe mode, and all of them when the host shuts down.
type Teardown struct {
	mode      *Mode
	sups      *supervision.Supervisors
	stopper   *supervision.Stopper
	evictor   evictionTimer
	events    eventCloser
	listeners listenerControl
	health    healthRemover
	report    func(error)
}

// TeardownDeps are what a Teardown stops and closes.
type TeardownDeps struct {
	Mode        *Mode
	Supervisors *supervision.Supervisors
	Stopper     *supervision.Stopper
	Evictor     *supervision.Evictor
	Events      eventCloser
	Listeners   listenerControl
	Health      healthRemover
	Report      func(error)
}

func NewTeardown(d TeardownDeps) *Teardown {
	return &Teardown{
		mode: d.Mode, sups: d.Supervisors, stopper: d.Stopper, evictor: d.Evictor, events: d.Events, listeners: d.Listeners,
		health: d.Health, report: d.Report,
	}
}

// StopPlugin stops the process of the plugin, if one runs, and forgets its endpoint.
func (t *Teardown) StopPlugin(ctx context.Context, pluginID string) error {
	return t.stopper.StopPlugin(ctx, pluginID)
}

// EvictIdle stops the plugins that hold no lease and have been idle for longer than their idle limit.
// defaultLimit is the limit of a plugin whose manifest sets none.
func (t *Teardown) EvictIdle(ctx context.Context, defaultLimit, gracePeriod time.Duration) []string {
	return t.stopper.EvictIdle(ctx, defaultLimit, gracePeriod)
}

// ReleaseProcess stops the process of the plugin at once where it holds no lease, and else at the first
// idle check after it holds none. A lease taken afterwards cancels the release.
func (t *Teardown) ReleaseProcess(ctx context.Context, pluginID string) {
	t.stopper.ReleaseProcess(ctx, pluginID)
}

// SetSafeMode enables or disables safe recovery mode. Entering it stops every running plugin and takes
// the health items of every plugin out of their list.
func (t *Teardown) SetSafeMode(enabled bool) {
	t.mode.setSafe(enabled)
	if enabled {
		t.health.RemoveAll()
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
	t.evictor.Stop()

	firstErr := t.events.closeBus(ctx)

	t.mode.close()
	targets := t.stopper.TakeRunning()
	t.listeners.closeAll()

	if err := t.stopper.StopTargets(ctx, targets, gracePeriod); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// IsEvicting reports whether the idle plugins are stopped on a timer.
func (t *Teardown) IsEvicting() bool {
	return t.evictor.IsRunning()
}
