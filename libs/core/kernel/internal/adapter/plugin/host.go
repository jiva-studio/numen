package plugin

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agenttoolsbridge"
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
	Storage      *Storage
	Events       *Events
	Mode         *Mode
}

// toolHost is what the catalog of tools starts plugins and lists them through.
type toolHost struct {
	*Activation
	*Registry
}

// NewHost creates a new plugin host for the specified target application and search paths.
// The host reads the time from clock.
func NewHost(clock machineport.Clock, target string, dirs ...string) *Host {
	t := newTable()
	errs := &errorSink{}
	failures := newFailures(t, clock, errs)
	services := newServices(t)
	storage := newStorage(t, errs)
	contentTypes := newContentTypes(t)
	processes := newProcesses(t, clock, errs, services, storage)
	listeners := newListeners(t, clock, errs, failures)
	settings := newSettings(t, errs, listeners)
	config := newConfigPush(t, errs, settings)
	settings.config = config
	registry := newRegistry(t, failures, contentTypes)
	activation := newActivation(t, clock, failures, processes)
	teardown := newTeardown(t, clock, errs, processes, services)
	discovery := newDiscovery(t, clock, errs, target, dirs)
	events := newEvents(t, clock, activation)
	mode := newMode(t, errs, teardown, listeners)

	activation.config, activation.discovery, activation.refreshListeners = config, discovery, listeners.RefreshListeners
	listeners.activation = activation
	teardown.listeners, teardown.events = listeners, events
	discovery.teardown, discovery.listeners, discovery.storage, discovery.config = teardown, listeners, storage, config
	discovery.dropEndpoint, processes.dropEndpoint = teardown.dropEndpoint, teardown.dropEndpoint
	config.prune = discovery.pruneSecretPermissions

	services.tools = func() hostendpoint.PluginTools {
		return agenttoolsbridge.NewCatalog(toolHost{Activation: activation, Registry: registry}, agenttoolsbridge.Full)
	}
	services.callers = hostendpoint.Callers{
		Identity: activation, Leases: activation, Completion: services, Vault: services, Texts: services,
		Search: services, Window: services, Work: services, Dependencies: services, Tools: services,
	}
	return &Host{
		Registry: registry, Discovery: discovery, Activation: activation, Processes: processes, Teardown: teardown,
		Failures: failures, Settings: settings, Config: config, Listeners: listeners, Services: services,
		ContentTypes: contentTypes, Storage: storage, Events: events, Mode: mode,
	}
}
