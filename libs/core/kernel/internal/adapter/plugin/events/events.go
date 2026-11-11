// Package events sends the events of the application to the plugins that declare an activation event for them.
package events

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/pluginset"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/eventbus"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// pluginActivator starts a plugin and counts the callers that hold it.
type pluginActivator interface {
	EnsureActive(ctx context.Context, pluginID string) (transport.Endpoint, error)
	Release(pluginID string)
}

// modeState is what the events read of the mode of the host.
type modeState interface {
	IsSafeMode() bool
}

// Events sends the events of the application to the plugins that declare an activation event for them.
type Events struct {
	set       *pluginset.Set
	mode      modeState
	sups      *supervision.Supervisors
	activator pluginActivator
	bus       *eventbus.Bus
}

// New returns the events over the set of plugins, the mode of the host, the supervisors and the activator.
func New(set *pluginset.Set, mode modeState, sups *supervision.Supervisors, activator pluginActivator, clock machineport.Clock) *Events {
	e := &Events{set: set, mode: mode, sups: sups, activator: activator}
	e.bus = eventbus.New(e, clock, 50*time.Millisecond, 250*time.Millisecond)
	return e
}

// EmitEvent queues an event into the debounced and throttled event bus.
func (e *Events) EmitEvent(event string) {
	if e.mode.IsSafeMode() || e.bus == nil {
		return
	}
	e.bus.Publish(event)
}

// GetEventBus returns the event bus attached to this host.
func (e *Events) GetEventBus() *eventbus.Bus {
	return e.bus
}

// CloseBus stops the delivery of the events.
func (e *Events) CloseBus(ctx context.Context) error {
	if e.bus == nil {
		return nil
	}
	return e.bus.Close(ctx)
}

func (e *Events) listActivatedBy(event string) []string {
	v, release := e.set.View()
	defer release()
	candidates := make([]string, 0)
	for _, p := range v.Enabled() {
		if p.Manifest.IsActivatedBy(event) {
			candidates = append(candidates, p.Manifest.ID)
		}
	}
	return candidates
}

// TriggerEvent activates every enabled plugin whose manifest declares an activation event matching event, and delivers the event to it.
func (e *Events) TriggerEvent(ctx context.Context, event string) ([]string, error) {
	if e.mode.IsSafeMode() {
		return nil, nil
	}
	candidates := e.listActivatedBy(event)

	activated := make([]string, 0, len(candidates))
	var errs []error
	for _, id := range candidates {
		endpoint, err := e.activator.EnsureActive(ctx, id)
		if err != nil {
			errs = append(errs, fmt.Errorf("activate plugin %s on %s: %w", id, event, err))
			continue
		}
		deliverErr := transport.DeliverPluginEvent(ctx, e.sups.Clients().Get(endpoint), endpoint, event)
		e.activator.Release(id)
		activated = append(activated, id)
		if deliverErr != nil {
			errs = append(errs, fmt.Errorf("deliver %s to plugin %s: %w", event, id, deliverErr))
		}
	}
	return activated, errors.Join(errs...)
}
