// Package pluginhost opens a plugin host the way the container does, and hands out its roles for a test that needs
// more than the capabilities the container lends.
package pluginhost

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/onnxruntime"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Host is the roles of a plugin host. It has the fields of the roles that the container holds, so that a test converts
// it with (*container.PluginRoles)(host) to lend it to the container.
type Host struct {
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

// Open returns a plugin host for the target that has discovered the plugins in dirs.
func Open(tb testing.TB, target string, dirs ...string) *Host {
	tb.Helper()
	host := New(clock.System{}, target, dirs...)
	host.ContentTypes.SetContentTypeTable(contenttype.Open())
	host.Services.SetDependencies(dependencies())
	if err := host.Discovery.Discover(); err != nil {
		tb.Logf("discover plugins: %v", err)
	}
	return host
}

// New creates the roles of a plugin host for the target application and the folders its plugins are in, as the
// container makes them. The roles read the time from clock.
func New(clock machineport.Clock, target string, dirs ...string) *Host {
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
	return &Host{
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

func dependencies() *provision.Registry {
	runtimeConfig := onnxruntime.FromEnvironment(clock.System{})
	runtimeConfig.Dir = onnxruntime.GetModelsDir(runtimeConfig.Dir)
	return provision.NewRegistry(
		onnxruntime.NewONNXProvisioner(runtimeConfig),
		provision.NewFileProvisioner(provision.Config{Dir: runtimeConfig.Dir, Offline: onnxruntime.IsOffline(), Clock: clock.System{}}),
	)
}

// Roles presents the roles of a host as one value, for a test that gives a bridge the whole host.
type Roles struct {
	*plugin.Activation
	*plugin.Registry
	*plugin.Settings
	*plugin.ConfigPush
	*supervision.Failures
	*plugin.Events
	*plugin.Mode
	*plugin.ContentTypes
	*plugin.Discovery
	*supervision.Services
}

// Join presents the roles of host as one value.
func Join(host *Host) Roles {
	return Roles{
		Activation: host.Activation, Registry: host.Registry, Settings: host.Settings, ConfigPush: host.Config,
		Failures: host.Failures, Events: host.Events, Mode: host.Mode, ContentTypes: host.ContentTypes,
		Discovery: host.Discovery, Services: host.Services,
	}
}
