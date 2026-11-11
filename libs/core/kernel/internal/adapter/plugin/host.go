package plugin

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/activation"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/configpush"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/discovery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/events"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hostmode"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/listeners"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/pluginset"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/teardown"
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
	Registry     *registry.Registry
	Discovery    *discovery.Discovery
	Activation   *activation.Activation
	Retry        *activation.Retry
	Processes    *supervision.Processes
	Teardown     *teardown.Teardown
	Failures     *supervision.Failures
	Settings     *settings.Settings
	Config       *configpush.ConfigPush
	Listeners    *listeners.Listeners
	Services     *supervision.Services
	ContentTypes *registry.ContentTypes
	Storage      *storage.Storage
	Events       *events.Events
	Mode         *hostmode.Mode
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
	host, _ := assemble(clock, target, dirs)
	return host
}

// assembly is the parts the roles of a host share, which the tests of this package reach.
type assembly struct {
	sups    *supervision.Supervisors
	set     *pluginset.Set
	scan    *pluginset.ScanLock
	stopper *supervision.Stopper
	evictor *supervision.Evictor
}

func assemble(clock machineport.Clock, target string, dirs []string) (*Host, assembly) {
	errs := &errsink.Sink{}
	mode := hostmode.New()
	values := pluginset.NewValues()
	readers := &contenttypes.Lookup{}

	services := supervision.NewServices(mode)
	sups := supervision.NewSupervisors(errs.Report, services.ClearPlugin)
	failures := supervision.NewFailures(sups, clock, errs)
	set := pluginset.New(sups, values, mode)
	scan := &pluginset.ScanLock{}
	files := storage.New(errs.Report, func(pluginID string) string {
		p, _ := set.GetKnown(pluginID)
		return p.Manifest.Version
	})
	processes := supervision.NewProcesses(sups, clock, errs.Report, services, files)
	stopper := supervision.NewStopper(sups, processes, services, clock, errs.Report)
	evictor := supervision.NewEvictor(stopper)

	config := configpush.New(set, sups, errs.Report)
	activator := activation.New(set, scan, mode, sups, processes, failures, config, clock)
	bus := events.New(set, mode, sups, activator, clock)
	listening := listeners.New(set, mode, sups, activator, failures, clock, errs.Report)

	host := &Host{
		Registry:     registry.New(set, mode, sups, failures, readers),
		Discovery:    discovery.New(set, scan, mode, sups, stopper, listening, files, config, errs.Report, dirs, target),
		Activation:   activator,
		Retry:        activation.NewRetry(set, failures, activator, listening),
		Processes:    processes,
		Teardown:     teardown.New(mode, sups, stopper, evictor, bus, listening, errs.Report),
		Failures:     failures,
		Settings:     settings.New(values, set, listening, config, errs.Report),
		Config:       config,
		Listeners:    listening,
		Services:     services,
		ContentTypes: registry.NewContentTypes(set, readers),
		Storage:      files,
		Events:       bus,
		Mode:         mode,
	}
	return host, assembly{sups: sups, set: set, scan: scan, stopper: stopper, evictor: evictor}
}
