package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/toolbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginprocess"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/rowoptions"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// StopPlugins stops every plugin the host runs.
func (c Config) StopPlugins(ctx context.Context) error {
	return stopPlugins(plugins(c), ctx)
}

func closeAfterWork(ctx context.Context, work *run.PluginWork, closeIndex func() error) func() error {
	return func() error {
		return errors.Join(work.WaitIdle(ctx), closeIndex())
	}
}

func openPluginWork(
	c Config, db *Index, v domain.Vault, tasks *run.Tasks, models *run.Lock,
) (pluginsWiring, *run.PluginWork, error) {
	setPluginVaultPath(plugins(c), v.Path)
	a := assemble(c, newTextVersions(vaultOf(c), db))
	if err := installCurrentVault(a, c, db, v); err != nil {
		return pluginsWiring{}, nil, err
	}
	work := pluginsOpenPluginWork(a, tasks, models)
	setPluginTexts(a.vault)
	return a, work, nil
}

func installCurrentVault(a pluginsWiring, c Config, db *Index, v domain.Vault) error {
	registry, err := c.registry()
	if err != nil {
		return err
	}
	current := func() (domain.Vault, bool) { return v, true }
	return installPluginVault(a, registry, db, current, PluginVaultHooks{})
}

// GetPluginServices is the forwarding of the services plugins serve to the windows.
func GetPluginServices(c Config) editor.PluginServices {
	return getPluginServices(plugins(c))
}

// InstallPluginVault lets the plugins work through the host the vaults of the installation and the index that
// holds them: the one current names, else the one a call names by identity.
func InstallPluginVault(c Config,
	db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) error {
	vaults, err := c.registry()
	if err != nil {
		return err
	}
	return installPluginVault(plugins(c), vaults, db, current, hooks)
}

// PluginRuntime is what runs, finds and addresses the plugins.
type PluginRuntime struct {
	bridge.Activator
	pluginport.PluginRetrier
	pluginport.PluginDiscoverer
	pluginport.PluginLister
	pluginport.PluginResolver
	pluginport.PluginEventEmitter
	pluginport.PluginBinding
	transport.ClientLookup
	textport.TextPlugins
	pluginport.PluginFailures
	pluginport.ConfigWatcher
}

// PluginConfiguration is what carries the settings to the plugins.
type PluginConfiguration struct {
	pluginport.PluginSettingDeclarer
	pluginport.ConfigRefresher
	pluginport.SettingsPush
	pluginport.SettingChecker
	pluginport.PersistHook
}

// PluginWatch is what ends the plugins, reports their dependencies and names the content types they give.
type PluginWatch struct {
	pluginport.PluginShutdown
	pluginport.DependencyReporter
	contenttypeport.ContentTypeTable
}

// PluginViews are the parts of the host that the settings take as they are, and that stay unset where the host runs
// nothing: the settings then accept every plugin value and list no plugin, and no connection type exists. Follow returns the
// subscription to the setting changes that the host follows, or none.
type PluginViews struct {
	Types      connectionport.Types
	Operations connectionport.OperationTitles
	Infos      settingsport.PluginInfos
	Checks     settingsport.PluginSettings
	Follow     func(subscribe settingsport.ChangeSubscriber) settingsport.ChangeSubscriber
}

// PluginHost is the plugin host as the capabilities the wiring and the use cases consume, each taken from the
// role of the host that does it. The settings subscription is keyed by the one *PluginHost the container
// made. ApplySetting makes a plugin follow one setting that changed. SetStorage sets the folders the plugins
// store files in and keeps the dependencies in the models folder the host names; RefreshDependencies
// provisions them as the environment names them now. Detection and RowOptions keep what the providers
// reported, for every window of the host, in the cache files of the plugins; RunningProviders opens the
// provider of a plugin only while it runs.
// SourceDescriptions keeps the description of the search source between its openings and processes.
type PluginHost struct {
	Runtime             PluginRuntime
	Configuration       PluginConfiguration
	Watch               PluginWatch
	Endpoint            *hostendpoint.Services
	ApplySetting        settingsport.PluginSettingApplier
	Views               PluginViews
	SetStorage          func(cacheRoot, dataRoot string)
	RefreshDependencies func()
	Health              *pluginhealth.Items
	Processes           pluginprocess.Processes
	Detection           *providerbridge.Detection
	RowOptions          *rowoptions.Lists
	RunningProviders    *providerbridge.RunningSource
	SourceDescriptions  *sourcebridge.Descriptions
}

// AdaptPluginHost lends the capabilities of host, the roles of a plugin host the caller made with NewPluginRoles, to the container. The
// tests of packages outside the container use it to run a window over a host they hold.
func AdaptPluginHost(host *PluginRoles) *PluginHost {
	return adaptPluginHost(host)
}

func adaptPluginHost(host *PluginRoles) *PluginHost {
	types := plugin.NewConnectionTypes(host.Registry)
	detection := providerbridge.NewDetection(types, providerbridge.NewQuestionSource(host.Activation), clock.System{})
	host.Activation.SetOnStart(detection.DetectInBackground)
	return &PluginHost{
		Runtime: PluginRuntime{
			Activator:          host.Activation,
			PluginRetrier:      host.Retry,
			PluginDiscoverer:   host.Discovery,
			PluginLister:       host.Registry,
			PluginResolver:     host.Registry,
			PluginEventEmitter: host.Events,
			PluginBinding:      plugin.NewBinding(host.Teardown, host.Mode),
			ClientLookup:       host.Activation.ClientLookup(),
			TextPlugins:        host.Registry,
			PluginFailures:     host.Failures,
			ConfigWatcher:      host.Config,
		},
		Configuration: PluginConfiguration{
			PluginSettingDeclarer: host.Registry,
			ConfigRefresher:       host.Config,
			SettingsPush:          host.Settings,
			SettingChecker:        host.Settings,
			PersistHook:           host.Settings,
		},
		Watch: PluginWatch{
			PluginShutdown:     host.Teardown,
			DependencyReporter: host.Services,
			ContentTypeTable:   host.ContentTypes,
		},
		Endpoint: lendHostEndpoint(host),
		ApplySetting: settingsusecase.NewApplyPluginSetting(settingsusecase.ApplyPluginSettingDeps{
			Values: host.Settings, Stopper: host.Teardown, Failures: host.Failures, Listeners: host.Listeners, Push: host.Config,
			Health: host.Health,
		}).Execute,
		Views: PluginViews{
			Types: types, Operations: types,
			Infos: host.Registry, Checks: host.Settings,
			Follow: func(subscribe settingsport.ChangeSubscriber) settingsport.ChangeSubscriber { return subscribe },
		},
		SetStorage: func(cacheRoot, dataRoot string) {
			setPluginStorage(host, cacheRoot, dataRoot, newDependencyRegistry)
			detection.SetFiles(host.Storage)
		},
		RefreshDependencies: func() { host.Services.SetDependencies(newDependencyRegistry("")) },
		Health:              host.Health,
		Processes:           host.RunningProcesses,
		Detection:           detection,
		RowOptions:          rowoptions.NewLists(host.Storage),
		RunningProviders:    providerbridge.NewRunningSource(host.Activation, host.Activation.ClientLookup()),
		SourceDescriptions:  sourcebridge.NewDescriptions(host.Storage, host.Config),
	}
}

func lendHostEndpoint(host *PluginRoles) *hostendpoint.Services {
	services := hostendpoint.NewServices()
	tools := toolbridge.NewCatalog(plugin.NewToolHost(host.Activation, host.Registry), host.Storage)
	host.Services.SetHostEndpoint(hostendpoint.NewServer(hostendpoint.Address, hostendpoint.Callers{
		Identity: host.Activation, Leases: host.Activation, Processes: host.Teardown,
		Lent: services,
		Tools: func() hostendpoint.PluginTools {
			return tools
		},
		Health: hostendpoint.HealthCalls{
			Report: pluginhealth.NewReport(host.Health, host.Registry),
			Clear:  pluginhealth.NewClear(host.Health),
		},
	}))
	return services
}
