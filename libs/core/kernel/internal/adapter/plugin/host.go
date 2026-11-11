package plugin

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/contenttypes"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/errsink"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Host manages the discovery, lifecycle, activation, and communication endpoints of plugins. It is
// the roles that do this: each keeps its own state and is made from the roles it needs, so that
// none reaches back to one made after it.
type Host struct {
	Registry     *Registry
	Discovery    *Discovery
	Activation   *Activation
	Retry        *Retry
	Processes    *supervision.Processes
	Teardown     *Teardown
	Failures     *supervision.Failures
	Settings     *Settings
	Config       *ConfigPush
	Listeners    *Listeners
	Services     *supervision.Services
	ContentTypes *ContentTypes
	Storage      *storage.Storage
	Events       *Events
	Mode         *Mode
}

// toolActivator starts a plugin for the catalog of tools and holds it.
type toolActivator interface {
	EnsureActive(ctx context.Context, pluginID string) (transport.Endpoint, error)
	Release(pluginID string)
}

// pluginInfoLister lists the plugins the host knows.
type pluginInfoLister interface {
	GetPluginInfos() []plugindomain.Info
}

// ToolHost is what the catalog of tools starts plugins and lists them through.
type ToolHost struct {
	activation toolActivator
	registry   pluginInfoLister
}

// NewToolHost returns the host the catalog of tools works over.
func NewToolHost(activation toolActivator, registry pluginInfoLister) ToolHost {
	return ToolHost{activation: activation, registry: registry}
}

// EnsureActive starts the plugin if it is not running and holds it.
func (h ToolHost) EnsureActive(ctx context.Context, pluginID string) (transport.Endpoint, error) {
	return h.activation.EnsureActive(ctx, pluginID)
}

// Release gives up the hold EnsureActive took.
func (h ToolHost) Release(pluginID string) { h.activation.Release(pluginID) }

// GetPluginInfos lists the plugins the host knows.
func (h ToolHost) GetPluginInfos() []plugindomain.Info { return h.registry.GetPluginInfos() }

// NewHost creates a new plugin host for the specified target application and search paths.
// The host reads the time from clock.
func NewHost(clock machineport.Clock, target string, dirs ...string) *Host {
	errs := &errsink.Sink{}
	mode := newMode()
	values := newSettingValues()
	readers := &contenttypes.Lookup{}

	services := supervision.NewServices(mode)
	sups := supervision.NewSupervisors(errs.Report, services.ClearPlugin)
	failures := supervision.NewFailures(sups, clock, errs)
	set := newPluginSet(sups, values, mode)
	registry := newRegistry(set, mode, sups, failures, readers)
	files := storage.New(errs.Report, func(pluginID string) string {
		p, _ := set.getKnown(pluginID)
		return p.Manifest.Version
	})
	processes := supervision.NewProcesses(sups, clock, errs.Report, services, files)
	stopper := supervision.NewStopper(sups, processes, services, clock, errs.Report)
	evictor := supervision.NewEvictor(stopper)

	config := newConfigPush(set, sups, errs.Report)
	scan := &scanGate{}
	activation := &Activation{
		set: set, scan: scan, mode: mode, sups: sups, processes: processes, failures: failures, config: config, clock: clock,
	}
	events := newEvents(set, mode, sups, activation, clock)
	listeners := &Listeners{
		set: set, mode: mode, sups: sups, activator: activation, failures: failures, clock: clock,
		report: errs.Report, bindings: make(map[string]map[string]listenerBinding),
	}

	return &Host{
		Registry: registry,
		Discovery: &Discovery{
			set: set, registrar: set, scan: scan, mode: mode, sups: sups, stopper: stopper, listeners: listeners,
			storage: files, scans: config, report: errs.Report, dirs: dirs, target: target,
		},
		Activation: activation,
		Retry:      newRetry(set, failures, activation, listeners),
		Processes:  processes,
		Teardown: &Teardown{
			mode: mode, sups: sups, stopper: stopper, evictor: evictor, events: events, listeners: listeners, report: errs.Report,
		},
		Failures:     failures,
		Settings:     newSettings(values, set, listeners, config, errs.Report),
		Config:       config,
		Listeners:    listeners,
		Services:     services,
		ContentTypes: newContentTypes(set, readers),
		Storage:      files,
		Events:       events,
		Mode:         mode,
	}
}
