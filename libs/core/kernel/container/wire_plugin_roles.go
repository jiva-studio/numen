package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
)

// PluginRoles are the roles of a plugin host, each keeping its own state and made from the roles it needs, so that
// none reaches back to one made after it. NewPluginRoles is the one place that makes them, for the container and for
// the tests that hold a host.
type PluginRoles struct {
	PluginSupervision
	Registry     *plugin.Registry
	Discovery    *plugin.Discovery
	Activation   *plugin.Activation
	Retry        *plugin.Retry
	Teardown     *plugin.Teardown
	Settings     *plugin.Settings
	Config       *plugin.ConfigPush
	Listeners    *plugin.Listeners
	ContentTypes *plugin.ContentTypes
	Storage      *storage.Storage
	Events       *plugin.Events
	Mode         *plugin.Mode
	Health       *pluginhealth.Items
}

// PluginSupervision are the roles that run the processes of the plugins: their start and stop, their
// failures and the services they serve, and the list of the ones that run.
type PluginSupervision struct {
	RunningProcesses *supervision.RunningProcesses
	Processes        *supervision.Processes
	Failures         *supervision.Failures
	Services         *supervision.Services
}

// NewPluginRoles creates the roles of a plugin host for the target application and the folders its plugins are in. The
// roles read the time from clock.
func NewPluginRoles(clock machineport.Clock, target string, dirs ...string) *PluginRoles {
	errs := plugin.NewErrorSink()
	mode := plugin.NewMode()
	values := plugin.NewSettingValues()
	readers := plugin.NewContentTypeLookup()
	services := supervision.NewServices(mode)
	sups := supervision.NewSupervisors(errs.Report, services.ClearPlugin)
	health := pluginhealth.NewItems()
	failures := supervision.NewFailures(sups, clock, errs, health)
	set := plugin.NewPluginSet(plugin.PluginSetDeps{Supervisors: sups, Values: values, Mode: mode})
	registry := plugin.NewRegistry(plugin.RegistryDeps{Set: set, Mode: mode, Supervisors: sups, Readers: readers})
	files := storage.New(errs.Report, set.GetVersion)
	processes := supervision.NewProcesses(sups, clock, errs.Report, services, files)
	stopper := supervision.NewStopper(sups, processes, services, clock, errs.Report)
	evictor := supervision.NewEvictor(stopper)
	config := plugin.NewConfigPush(plugin.ConfigPushDeps{Set: set, Supervisors: sups, Report: errs.Report, Health: health})
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
		PluginSupervision: PluginSupervision{RunningProcesses: supervision.NewRunningProcesses(sups), Processes: processes, Failures: failures, Services: services},
		Registry:          registry,
		Discovery: plugin.NewDiscovery(plugin.DiscoveryDeps{
			Set: set, Registrar: set, Scan: scan, Mode: mode, Supervisors: sups, Stopper: stopper, Listeners: listeners, Storage: files,
			Health: health, Report: errs.Report, Dirs: dirs, Target: target,
		}),
		Activation: activation,
		Retry:      plugin.NewRetry(plugin.RetryDeps{Set: set, Failures: failures, Activation: activation, Listeners: listeners}),
		Teardown: plugin.NewTeardown(plugin.TeardownDeps{
			Mode: mode, Supervisors: sups, Stopper: stopper, Evictor: evictor, Events: events, Listeners: listeners, Health: health, Report: errs.Report,
		}),
		Settings: plugin.NewSettings(plugin.SettingsDeps{
			Values: values, Set: set, Listeners: listeners, Profile: config, Report: errs.Report,
		}),
		Config:       config,
		Listeners:    listeners,
		ContentTypes: plugin.NewContentTypes(plugin.ContentTypesDeps{Set: set, Lookup: readers}),
		Storage:      files,
		Events:       events,
		Mode:         mode,
		Health:       health,
	}
}

func setPluginStorage(roles *PluginRoles, cacheRoot, dataRoot string, newRegistry func(modelsDir string) pluginport.DependencyRegistry) {
	roles.Storage.SetStorageRoots(cacheRoot, dataRoot)
	// gomutants:disable-next-line reason="a window started without a models directory gets no dependency registry, which only an installation with downloaded models exercises"
	if models := roles.Storage.GetModelsDir(); models != "" {
		// gomutants:disable-next-line reason="the registry is built from the models directory of an installation, which a test does not have"
		roles.Services.SetDependencies(newRegistry(models))
	}
}
