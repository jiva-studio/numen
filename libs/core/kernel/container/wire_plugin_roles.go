package container

import (
	pluginsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/toolbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
)

// PluginManagement provides discovery, activation and settings for installed plugins.
type PluginManagement struct {
	Registry     *plugin.Registry
	Discovery    *plugin.Discovery
	Activation   *plugin.Activation
	Retry        *plugin.Retry
	Teardown     *plugin.Teardown
	Settings     *plugin.Settings
	Config       *plugin.ConfigPush
	Listeners    *plugin.Listeners
	ContentTypes *plugin.ContentTypes
	Events       *plugin.Events
	Mode         *plugin.Mode
}

// PluginSupervision owns the process, storage and service dependencies of the plugin runtime.
type PluginSupervision struct {
	RunningProcesses *supervision.RunningProcesses
	Processes        *supervision.Processes
	Failures         *supervision.Failures
	Services         *supervision.Services
	Endpoint         *supervision.Slot[supervision.HostEndpoint]
	Dependencies     *supervision.Slot[supervision.Dependencies]
	Storage          *storage.Storage
	Health           *pluginhealth.Items
	Clock            machineport.Clock
}

// NewPluginRoles shares one supervisor set and activation mode across discovery, execution and teardown.
func NewPluginRoles(clock machineport.Clock, target string, dirs ...string) (*PluginManagement, *PluginSupervision) {
	errs := plugin.NewErrorSink()
	mode := plugin.NewMode()
	values := plugin.NewSettingValues()
	readers := plugin.NewContentTypeLookup()
	endpoint, dependencies := &supervision.Slot[supervision.HostEndpoint]{}, &supervision.Slot[supervision.Dependencies]{}
	services := supervision.NewServices(mode, endpoint.Get, dependencies.Get)
	sups := supervision.NewSupervisors(errs.Report, services.ClearPlugin)
	health := pluginhealth.NewItems()
	failures := supervision.NewFailures(sups, clock, errs, health)
	set := plugin.NewPluginSet(plugin.PluginSetDeps{Supervisors: sups, Values: values, Mode: mode, Core: getCoreClaims()})
	files := storage.New(errs.Report, set.GetVersion)
	registry := plugin.NewRegistry(plugin.RegistryDeps{Set: set, Mode: mode, Supervisors: sups, Readers: readers, GetTools: listKeptToolItems(files)})
	config := plugin.NewConfigPush(plugin.ConfigPushDeps{Set: set, Supervisors: sups, Report: errs.Report, Health: health})
	processes := supervision.NewProcesses(sups, clock, errs.Report, services, files, config.GetProcessNice)
	stopper := supervision.NewStopper(sups, processes, services, clock, errs.Report)
	evictor := supervision.NewEvictor(stopper)
	scan := plugin.NewScanGate()
	activation := plugin.NewActivation(plugin.ActivationDeps{
		Set: set, Enablement: registry, Scan: scan, Mode: mode, Supervisors: sups, Processes: processes, Failures: failures,
		Config: config, Clock: clock,
	})
	events := plugin.NewEvents(plugin.EventsDeps{
		Set: set, Mode: mode, Supervisors: sups, Activation: activation, Clock: clock, Report: errs.Report,
	})
	listeners := plugin.NewListeners(plugin.ListenersDeps{
		Set: set, Mode: mode, Supervisors: sups, Activation: activation, Failures: failures, Clock: clock, Report: errs.Report,
	})
	processHost := &PluginSupervision{
		RunningProcesses: supervision.NewRunningProcesses(sups), Processes: processes, Failures: failures, Services: services,
		Endpoint: endpoint, Dependencies: dependencies, Storage: files, Health: health, Clock: clock,
	}
	return &PluginManagement{
		Registry: registry,
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
		Events:       events,
		Mode:         mode,
	}, processHost
}

func setPluginStorage(roles *PluginSupervision, cacheRoot, dataRoot string, newRegistry func(modelsDir string) supervision.Dependencies) {
	roles.Storage.SetStorageRoots(cacheRoot, dataRoot)
	// gomutants:disable-next-line reason="a window started without a models directory gets no dependency registry, which only an installation with downloaded models exercises"
	if models := roles.Storage.GetModelsDir(); models != "" {
		// gomutants:disable-next-line reason="the registry is built from the models directory of an installation, which a test does not have"
		roles.Dependencies.Replace(newRegistry(models))
	}
}

func listKeptToolItems(files toolbridge.ListingFiles) func(pluginID string) []pluginsdomain.ContributionItem {
	return func(pluginID string) []pluginsdomain.ContributionItem {
		kept := toolbridge.ListKeptTools(files, pluginID)
		items := make([]pluginsdomain.ContributionItem, 0, len(kept))
		for _, one := range kept {
			text := one.GetTitle()
			if text == "" {
				text = one.GetName()
			}
			items = append(items, pluginsdomain.ContributionItem{Text: text, Detail: one.GetName()})
		}
		return items
	}
}
