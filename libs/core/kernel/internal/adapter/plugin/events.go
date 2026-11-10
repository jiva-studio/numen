package plugin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/eventbus"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Events sends the events of the application to the plugins that declare an activation event for them.
type Events struct {
	t          *table
	activation *Activation
	bus        *eventbus.Bus
}

func newEvents(t *table, clock machineport.Clock, activation *Activation) *Events {
	e := &Events{t: t, activation: activation}
	e.bus = eventbus.New(e, clock, 50*time.Millisecond, 250*time.Millisecond)
	return e
}

// EmitEvent queues an event into the debounced and throttled event bus.
func (e *Events) EmitEvent(event string) {
	e.t.mu.RLock()
	isSafe := e.t.safeMode
	e.t.mu.RUnlock()

	if isSafe || e.bus == nil {
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

// TriggerEvent activates every enabled plugin whose manifest declares an activation event matching event, and delivers the event to it.
func (e *Events) TriggerEvent(ctx context.Context, event string) ([]string, error) {
	e.t.mu.RLock()
	if e.t.safeMode {
		e.t.mu.RUnlock()
		return nil, nil
	}
	candidates := make([]string, 0)
	for _, p := range e.t.getPluginsInOrderLocked() {
		id := p.Manifest.ID
		if !e.t.isPluginEnabledLocked(id) {
			continue
		}
		if p.Manifest.IsActivatedBy(event) {
			candidates = append(candidates, id)
		}
	}
	e.t.mu.RUnlock()

	activated := make([]string, 0, len(candidates))
	var errs []error
	for _, id := range candidates {
		endpoint, err := e.activation.EnsureActive(ctx, id)
		if err != nil {
			errs = append(errs, fmt.Errorf("activate plugin %s on %s: %w", id, event, err))
			continue
		}
		deliverErr := transport.DeliverPluginEvent(ctx, e.t.clients.Get(endpoint), endpoint, event)
		e.activation.Release(id)
		activated = append(activated, id)
		if deliverErr != nil {
			errs = append(errs, fmt.Errorf("deliver %s to plugin %s: %w", event, id, deliverErr))
		}
	}
	return activated, errors.Join(errs...)
}
