package plugin

import (
	"context"
	"errors"
	"fmt"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/eventbus"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// pluginViews reads the set of plugins.
type pluginViews interface {
	view() (setView, func())
}

// Events sends the events of the application to the plugins that declare an activation event for them.
type Events struct {
	set       pluginViews
	mode      safeModeReader
	sups      *supervision.Supervisors
	activator pluginActivator
	bus       *eventbus.Bus
}

// EventsDeps are what Events reads and starts, and the clock that times the event bus.
type EventsDeps struct {
	Set         pluginViews
	Mode        safeModeReader
	Supervisors *supervision.Supervisors
	Activation  pluginActivator
	Clock       machineport.Clock
}

// NewEvents creates the events and the bus that debounces and throttles them.
func NewEvents(d EventsDeps) *Events {
	e := &Events{set: d.Set, mode: d.Mode, sups: d.Supervisors, activator: d.Activation}
	e.bus = eventbus.New(e, d.Clock, 50*time.Millisecond, 250*time.Millisecond)
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

func (e *Events) closeBus(ctx context.Context) error {
	if e.bus == nil {
		return nil
	}
	return e.bus.Close(ctx)
}

func (e *Events) listActivatedBy(event string) []string {
	v, release := e.set.view()
	defer release()
	candidates := make([]string, 0)
	for _, p := range v.enabled() {
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
		holder := plugindomain.GetHolder(ctx, "deliver event "+event)
		endpoint, err := e.activator.EnsureActive(ctx, id, holder)
		if err != nil {
			errs = append(errs, fmt.Errorf("activate plugin %s on %s: %w", id, event, err))
			continue
		}
		deliverErr := transport.DeliverPluginEvent(ctx, e.sups.Clients().Get(endpoint), endpoint, event)
		e.activator.Release(id, holder)
		activated = append(activated, id)
		if deliverErr != nil {
			errs = append(errs, fmt.Errorf("deliver %s to plugin %s: %w", event, id, deliverErr))
		}
	}
	return activated, errors.Join(errs...)
}
