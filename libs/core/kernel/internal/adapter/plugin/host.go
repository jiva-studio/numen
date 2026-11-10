package plugin

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Host manages the discovery, lifecycle, activation, and communication endpoints of plugins. It is
// the roles that do this, built over one table of plugins and wired to each other; it does
// nothing itself.
type Host struct {
	Registry     *Registry
	Discovery    *Discovery
	Activation   *Activation
	Processes    *Processes
	Teardown     *Teardown
	Failures     *Failures
	Settings     *Settings
	Config       *ConfigPush
	Listeners    *Listeners
	Services     *Services
	ContentTypes *ContentTypes
	Storage      *storage.Storage
	Events       *Events
	Mode         *Mode
}

// ToolHost is what the catalog of tools starts plugins and lists them through.
type ToolHost struct {
	activation *Activation
	registry   *Registry
}

// NewToolHost returns the host the catalog of tools works over.
func NewToolHost(activation *Activation, registry *Registry) ToolHost {
	return ToolHost{activation: activation, registry: registry}
}

// EnsureActive starts the plugin if it is not running and holds it.
func (h ToolHost) EnsureActive(ctx context.Context, pluginID string) (plugindomain.Endpoint, error) {
	return h.activation.EnsureActive(ctx, pluginID)
}

// Release gives up the hold EnsureActive took.
func (h ToolHost) Release(pluginID string) { h.activation.Release(pluginID) }

// GetPluginInfos lists the plugins the host knows.
func (h ToolHost) GetPluginInfos() []plugindomain.Info { return h.registry.GetPluginInfos() }

// NewHost creates a new plugin host for the specified target application and search paths.
// The host reads the time from clock.
func NewHost(clock machineport.Clock, target string, dirs ...string) *Host {
	t := newTable()
	errs := &errorSink{}
	failures := newFailures(t, clock, errs)
	services := newServices(t)
	storage := storage.New(errs.report, func(pluginID string) string {
		t.mu.RLock()
		defer t.mu.RUnlock()
		return t.plugins[pluginID].Manifest.Version
	})
	contentTypes := newContentTypes(t)
	processes := newProcesses(t, clock, errs, services, storage)
	listeners := newListeners(t, clock, errs, failures)
	settings := newSettings(t, errs, listeners)
	config := newConfigPush(t, errs, settings)
	settings.config = config
	registry := newRegistry(t, failures, contentTypes)
	activation := newActivation(t, clock, failures, processes)
	teardown := newTeardown(t, clock, errs, processes, services)
	discovery := newDiscovery(t, errs, target, dirs)
	events := newEvents(t, clock, activation)
	mode := newMode(t, errs, teardown, listeners)

	activation.config, activation.discovery, activation.refreshListeners = config, discovery, listeners.RefreshListeners
	listeners.activation = activation
	teardown.listeners, teardown.events = listeners, events
	discovery.teardown, discovery.listeners, discovery.storage, discovery.config = teardown, listeners, storage, config
	discovery.dropEndpoint, processes.dropEndpoint = teardown.dropEndpoint, teardown.dropEndpoint
	config.prune = discovery.pruneSecretPermissions

	return &Host{
		Registry: registry, Discovery: discovery, Activation: activation, Processes: processes, Teardown: teardown,
		Failures: failures, Settings: settings, Config: config, Listeners: listeners, Services: services,
		ContentTypes: contentTypes, Storage: storage, Events: events, Mode: mode,
	}
}
