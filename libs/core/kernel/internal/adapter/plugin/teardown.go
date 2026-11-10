package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

const (
	stopPluginGracePeriod = 2 * time.Second
	stopPluginTimeout     = 5 * time.Second
)

// idleThreshold is how long a plugin without a lease may stay idle before the host stops it.
const idleThreshold = 180 * time.Second

type pluginTeardownTarget struct {
	id       string
	pid      int
	sup      *lifecycle.Supervisor
	endpoint plugindomain.Endpoint
}

// Teardown stops the processes of the plugins: one on request, the idle ones on a timer, and all of
// them when the host shuts down.
type Teardown struct {
	t         *table
	clock     machineport.Clock
	errs      *errorSink
	processes *Processes
	services  *Services
	listeners *Listeners
	events    *Events

	evictTicker    *time.Ticker
	evictDone      chan struct{}
	evictWg        sync.WaitGroup
	isEvictStopped bool
}

func newTeardown(t *table, clock machineport.Clock, errs *errorSink, processes *Processes, services *Services) *Teardown {
	d := &Teardown{
		t: t, clock: clock, errs: errs, processes: processes, services: services,
		evictTicker: time.NewTicker(30 * time.Second),
		evictDone:   make(chan struct{}),
	}
	d.startEvictLoop()
	return d
}

// StopPlugin stops the process of the plugin, if one runs, and forgets its endpoint.
func (d *Teardown) StopPlugin(ctx context.Context, pluginID string) error {
	d.t.mu.Lock()
	sup, ok := d.t.supervisors[pluginID]
	if !ok {
		d.t.mu.Unlock()
		return nil
	}
	target := pluginTeardownTarget{id: pluginID, pid: sup.GetProcessID(), sup: sup, endpoint: d.t.endpoints[pluginID]}
	d.t.forgetEndpointLocked(pluginID)
	d.t.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, stopPluginTimeout)
	defer cancel()
	if err := d.teardownTarget(ctx, target, stopPluginGracePeriod); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
		return fmt.Errorf("stop plugin %s: %w", pluginID, err)
	}
	return nil
}

func (d *Teardown) teardownTarget(ctx context.Context, target pluginTeardownTarget, gracePeriod time.Duration) error {
	rpcShutdown := createRPCShutdown(target.endpoint, gracePeriod)
	err := target.sup.Teardown(ctx, gracePeriod, rpcShutdown)
	d.processes.closeStdin(target.id, target.pid)
	d.services.clearPlugin(target.id)
	return err
}

func (d *Teardown) dropEndpoint(pluginID string) {
	d.t.mu.Lock()
	d.t.forgetEndpointLocked(pluginID)
	d.t.mu.Unlock()
	d.services.clearPlugin(pluginID)
}

// EvictIdle stops the plugins that hold no lease and have been idle for longer than threshold.
func (d *Teardown) EvictIdle(ctx context.Context, threshold, gracePeriod time.Duration) []string {
	d.t.mu.Lock()
	now := d.clock.Now()
	var targets []pluginTeardownTarget

	for id, sup := range d.t.supervisors {
		pid := sup.GetProcessID()
		if sup.BeginDeactivation(threshold, now) {
			targets = append(targets, pluginTeardownTarget{
				id:       id,
				pid:      pid,
				sup:      sup,
				endpoint: d.t.endpoints[id],
			})
			d.t.forgetEndpointLocked(id)
		}
	}
	d.t.mu.Unlock()

	return d.evictTargets(ctx, targets, gracePeriod)
}

func (d *Teardown) evictTargets(ctx context.Context, targets []pluginTeardownTarget, gracePeriod time.Duration) []string {
	evicted := make([]string, 0, len(targets))
	for _, target := range targets {
		if err := d.teardownTarget(ctx, target, gracePeriod); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
			d.errs.report(fmt.Errorf("evict plugin %s: %w", target.id, err))
			continue
		}
		evicted = append(evicted, target.id)
	}
	return evicted
}

func (d *Teardown) startEvictLoop() {
	d.evictWg.Go(func() {
		for {
			select {
			case <-d.evictDone:
				return
			case <-d.evictTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = d.EvictIdle(ctx, idleThreshold, 2*time.Second)
				cancel()
			}
		}
	})
}

func (d *Teardown) stopEvictLoop() {
	d.t.mu.Lock()
	if !d.isEvictStopped {
		d.isEvictStopped = true
		if d.evictTicker != nil {
			d.evictTicker.Stop()
		}
		close(d.evictDone)
	}
	d.t.mu.Unlock()
	d.evictWg.Wait()
}

// TeardownAll shuts down all active plugin processes.
func (d *Teardown) TeardownAll(ctx context.Context, gracePeriod time.Duration) error {
	d.stopEvictLoop()

	var firstErr error
	if err := d.events.closeBus(ctx); err != nil {
		firstErr = err
	}

	targets := d.takeTeardownTargets()
	d.listeners.closeAll()

	for _, target := range targets {
		if err := d.teardownTarget(ctx, target, gracePeriod); err != nil && firstErr == nil &&
			!errors.Is(err, lifecycle.ErrProcessNotRunning) {
			firstErr = err
		}
	}
	if err := d.processes.closeAllStdins(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := d.services.closeHostEndpoint(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (d *Teardown) takeTeardownTargets() []pluginTeardownTarget {
	d.t.mu.Lock()
	defer d.t.mu.Unlock()
	d.t.isClosed = true
	var targets []pluginTeardownTarget
	for id, sup := range d.t.supervisors {
		if sup.GetState() != lifecycle.StateDormant {
			targets = append(targets, pluginTeardownTarget{
				id:       id,
				pid:      sup.GetProcessID(),
				sup:      sup,
				endpoint: d.t.endpoints[id],
			})
			d.t.forgetEndpointLocked(id)
		}
	}
	return targets
}
