package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/protocol"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/eventbus"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// pluginViews reads the set of plugins.
type pluginViews interface {
	getView() (setView, func())
}

// Events sends the events of the application to the plugins that declare an activation event for them.
type Events struct {
	set       pluginViews
	mode      safeModeReader
	sups      *supervision.Supervisors
	activator pluginActivator
	bus       *eventbus.Bus

	mu        sync.Mutex
	fileTypes map[string][]domain.FileType
}

// EventsDeps are what Events reads and starts, the clock that times the event bus, and where the failures of the
// deliveries that run after an event was emitted are reported.
type EventsDeps struct {
	Set         pluginViews
	Mode        safeModeReader
	Supervisors *supervision.Supervisors
	Activation  pluginActivator
	Clock       machineport.Clock
	Report      func(error)
}

// NewEvents creates the events and the bus that debounces and throttles them.
func NewEvents(d EventsDeps) *Events {
	e := &Events{set: d.Set, mode: d.Mode, sups: d.Supervisors, activator: d.Activation, fileTypes: map[string][]domain.FileType{}}
	e.bus = eventbus.New(e, d.Clock, 50*time.Millisecond, 250*time.Millisecond, d.Report)
	return e
}

// EmitEvent queues an event into the debounced and throttled event bus.
func (e *Events) EmitEvent(event string) {
	if e.mode.IsSafeMode() || e.bus == nil {
		return
	}
	e.bus.Publish(event)
}

// EmitFilesChanged queues the onFilesChanged event of the vault for the plugins that take a file of one of the
// types: a plugin whose manifest declares onFilesChanged with a mask takes the types the mask matches, and
// one that declares it bare takes every type. Types emitted before the event is delivered are delivered with it.
func (e *Events) EmitFilesChanged(vaultID string, types []domain.FileType) {
	if e.mode.IsSafeMode() || e.bus == nil {
		return
	}
	event := protocol.GetFilesChangedEvent(vaultID)
	e.mu.Lock()
	merged := slices.Concat(e.fileTypes[event], types)
	slices.SortFunc(merged, domain.CompareFileTypes)
	e.fileTypes[event] = slices.Compact(merged)
	e.mu.Unlock()
	e.bus.Publish(event)
}

func (e *Events) takeFileTypes(event string) ([]domain.FileType, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	types, isFiltered := e.fileTypes[event]
	delete(e.fileTypes, event)
	return types, isFiltered
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
	types, isFiltered := e.takeFileTypes(event)
	if _, isFilesChanged := protocol.ReadFilesChangedEvent(event); isFilesChanged && !isFiltered {
		return nil
	}
	v, release := e.set.getView()
	defer release()
	candidates := make([]string, 0)
	for _, p := range v.enabled() {
		if p.Manifest.IsActivatedBy(event) && (!isFiltered || p.Manifest.IsTakingFileTypes(types)) {
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
		holder := plugindomain.GetHolder(ctx, "handle event "+event)
		endpoint, err := e.activator.EnsureActive(withoutProvisioning(ctx), id, holder)
		if err != nil {
			errs = append(errs, fmt.Errorf("activate plugin %s on %s: %w", id, event, err))
			continue
		}
		deliverErr := transport.DeliverPluginEvent(ctx, e.sups.Clients().Get(endpoint), endpoint, event)
		e.activator.Release(id, endpoint.Token, holder)
		activated = append(activated, id)
		if deliverErr != nil {
			errs = append(errs, fmt.Errorf("deliver %s to plugin %s: %w", event, id, deliverErr))
		}
	}
	return activated, errors.Join(errs...)
}
