package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// PluginRoles are the roles of a plugin host, each keeping its own state and made from the roles it needs, so that
// none reaches back to one made after it.
type PluginRoles struct {
	Registry     *plugin.Registry
	Discovery    *plugin.Discovery
	Activation   *plugin.Activation
	Retry        *plugin.Retry
	Processes    *supervision.Processes
	Teardown     *plugin.Teardown
	Failures     *supervision.Failures
	Settings     *plugin.Settings
	Config       *plugin.ConfigPush
	Listeners    *plugin.Listeners
	Services     *supervision.Services
	ContentTypes *plugin.ContentTypes
	Storage      *storage.Storage
	Events       *plugin.Events
	Mode         *plugin.Mode
}

// newPluginRoles creates the roles of a plugin host for the target application and the folders its plugins are in.
// The roles read the time from clock.
func newPluginRoles(clock machineport.Clock, target string, dirs []string) *PluginRoles {
	errs := plugin.NewErrorSink()
	mode := plugin.NewMode()
	values := plugin.NewSettingValues()
	readers := plugin.NewContentTypeLookup()
	services := supervision.NewServices(mode)
	sups := supervision.NewSupervisors(errs.Report, services.ClearPlugin)
	failures := supervision.NewFailures(sups, clock, errs)
	set := plugin.NewPluginSet(plugin.PluginSetDeps{Supervisors: sups, Values: values, Mode: mode})
	registry := plugin.NewRegistry(plugin.RegistryDeps{Set: set, Mode: mode, Supervisors: sups, Failures: failures, Readers: readers})
	files := storage.New(errs.Report, set.GetVersion)
	processes := supervision.NewProcesses(sups, clock, errs.Report, services, files)
	stopper := supervision.NewStopper(sups, processes, services, clock, errs.Report)
	evictor := supervision.NewEvictor(stopper)
	config := plugin.NewConfigPush(plugin.ConfigPushDeps{Set: set, Supervisors: sups, Report: errs.Report})
	scan := plugin.NewScanGate()
	activation := plugin.NewActivation(plugin.ActivationDeps{
		Set: set, Enablement: registry, Scan: scan, Mode: mode, Supervisors: sups, Processes: processes, Failures: failures,
		Config: config, Clock: clock,
	})
	events := plugin.NewEvents(plugin.EventsDeps{Set: set, Mode: mode, Supervisors: sups, Activation: activation, Clock: clock})
	listeners := plugin.NewListeners(plugin.ListenersDeps{
		Set: set, Mode: mode, Supervisors: sups, Activation: activation, Failures: failures, Clock: clock, Report: errs.Report,
	})
	return &PluginRoles{
		Registry: registry,
		Discovery: plugin.NewDiscovery(plugin.DiscoveryDeps{
			Set: set, Registrar: set, Scan: scan, Mode: mode, Supervisors: sups, Stopper: stopper, Listeners: listeners, Storage: files,
			Config: config, Report: errs.Report, Dirs: dirs, Target: target,
		}),
		Activation: activation,
		Retry:      plugin.NewRetry(plugin.RetryDeps{Set: set, Failures: failures, Activation: activation, Listeners: listeners}),
		Processes:  processes,
		Teardown: plugin.NewTeardown(plugin.TeardownDeps{
			Mode: mode, Supervisors: sups, Stopper: stopper, Evictor: evictor, Events: events, Listeners: listeners, Report: errs.Report,
		}),
		Failures: failures,
		Settings: plugin.NewSettings(plugin.SettingsDeps{
			Values: values, Set: set, Listeners: listeners, Profile: config, Report: errs.Report,
		}),
		Config:       config,
		Listeners:    listeners,
		Services:     services,
		ContentTypes: plugin.NewContentTypes(plugin.ContentTypesDeps{Set: set, Lookup: readers}),
		Storage:      files,
		Events:       events,
		Mode:         mode,
	}
}

// setPluginStorage sets the folders the plugins store files in. Where the storage names a models folder, the
// dependencies of the plugins are provisioned into it by the registry newRegistry makes.
func setPluginStorage(roles *PluginRoles, cacheRoot, dataRoot string, newRegistry func(modelsDir string) pluginport.DependencyRegistry) {
	roles.Storage.SetStorageRoots(cacheRoot, dataRoot)
	// gomutants:disable-next-line reason="a window started without a models directory gets no dependency registry, which only an installation with downloaded models exercises"
	if models := roles.Storage.GetModelsDir(); models != "" {
		// gomutants:disable-next-line reason="the registry is built from the models directory of an installation, which a test does not have"
		roles.Services.SetDependencies(newRegistry(models))
	}
}
