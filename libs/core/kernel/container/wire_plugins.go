package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/toolbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/licenseport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginmanifest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginprocess"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/rowoptions"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func openPluginWork(
	install pluginInstall, vaultDeps vaultWiring, db *Index, v domain.Vault, tasks *run.Tasks, models *run.Lock,
) (*run.PluginWork, error) {
	setPluginVaultPath(install.Execution.Runtime.PluginBinding, v.Path)
	if err := installCurrentVault(install, vaultDeps, db, v); err != nil {
		return nil, err
	}
	work := pluginsOpenPluginWork(install, tasks, models)
	setPluginTexts(vaultDeps)
	return work, nil
}

func installCurrentVault(install pluginInstall, vaultDeps vaultWiring, db *Index, v domain.Vault) error {
	registry, err := vaultRegistry(vaultDeps.RegistryPath)
	if err != nil {
		return err
	}
	current := func() (domain.Vault, bool) { return v, true }
	return installPluginVault(install, vaultDeps, registry, db, current, PluginVaultHooks{})
}

// PluginRuntime is what runs, finds and addresses the plugins.
type PluginRuntime struct {
	bridge.Activator
	pluginport.PluginRetrier
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
	settingsport.PluginConfig
	settingsport.PluginSettings
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
	Types        connectionport.Types
	Operations   connectionport.OperationTitles
	Declarations licenseport.Declarations
	Infos        settingsport.PluginInfos
	Checks       settingsport.PluginSettings
	Follow       func(subscribe settingsport.ChangeSubscriber) settingsport.ChangeSubscriber
}

// PluginExecution runs plugins and tracks their processes.
type PluginExecution struct {
	Runtime   PluginRuntime
	Watch     PluginWatch
	Health    *pluginhealth.Items
	Processes pluginprocess.Processes
}

// PluginPreferences applies settings and storage locations to the installation's plugins.
type PluginPreferences struct {
	Configuration       PluginConfiguration
	Views               PluginViews
	ApplySetting        settingsport.PluginSettingApplier
	SetStorage          func(cacheRoot, dataRoot string)
	RefreshDependencies func()
	Detection           connectionport.Detected
}

// PluginProviders connects plugin providers to search and model clients.
type PluginProviders struct {
	Detection          *providerbridge.Detection
	RowOptions         *rowoptions.Lists
	RunningProviders   *providerbridge.RunningSource
	SourceDescriptions *sourcebridge.Descriptions
}

// PluginEndpoints connects the host RPC services and the providers served by plugins.
type PluginEndpoints struct {
	Endpoint *hostendpoint.Services
	Lender   *hostendpoint.Lender
	PluginProviders
}

// AdaptPluginHost connects the process and management roles to their consumers.
func AdaptPluginHost(management *PluginManagement, supervision *PluginSupervision) (*PluginExecution, *PluginPreferences, *PluginEndpoints) {
	lender := hostendpoint.NewLender()
	lent := lendHostEndpoint(management, supervision, lender)
	types := plugin.NewConnectionTypes(management.Registry)
	detection := providerbridge.NewDetection(types, providerbridge.NewQuestionSource(management.Activation), supervision.Clock)
	management.Activation.SetOnStart(detection.DetectInBackground)
	exec := &PluginExecution{
		Runtime: PluginRuntime{
			Activator:          management.Activation,
			PluginRetrier:      management.Retry,
			PluginLister:       management.Registry,
			PluginResolver:     management.Registry,
			PluginEventEmitter: management.Events,
			PluginBinding:      plugin.NewBinding(management.Teardown, management.Mode),
			ClientLookup:       management.Activation.ClientLookup(),
			TextPlugins:        management.Registry,
			PluginFailures:     supervision.Failures,
			ConfigWatcher:      management.Config,
		},
		Watch: PluginWatch{
			PluginShutdown:     management.Teardown,
			DependencyReporter: supervision.Services,
			ContentTypeTable:   management.ContentTypes,
		},
		Health:    supervision.Health,
		Processes: supervision.RunningProcesses,
	}
	prefs := &PluginPreferences{
		Configuration: PluginConfiguration{
			PluginSettingDeclarer: management.Registry,
			ConfigRefresher:       management.Config,
			PluginConfig:          management.Settings,
			PluginSettings:        management.Settings,
		},
		ApplySetting: settingsusecase.NewApplyPluginSetting(settingsusecase.ApplyPluginSettingDeps{
			Values: management.Settings, Stopper: management.Teardown, Failures: supervision.Failures, Listeners: management.Listeners, Push: management.Config,
			Health: supervision.Health,
		}).Execute,
		Views: PluginViews{
			Types: types, Operations: types,
			Declarations: management.Registry, Infos: management.Registry, Checks: management.Settings,
			Follow: func(subscribe settingsport.ChangeSubscriber) settingsport.ChangeSubscriber { return subscribe },
		},
		SetStorage: func(cacheRoot, dataRoot string) {
			setPluginStorage(supervision, cacheRoot, dataRoot, func(modelsDir string) pluginport.DependencyRegistry {
				return newDependencyRegistry(supervision.Clock, modelsDir)
			})
			detection.SetFiles(supervision.Storage)
		},
		RefreshDependencies: func() { supervision.Dependencies.Replace(newDependencyRegistry(supervision.Clock, "")) },
		Detection:           detection,
	}
	endpoints := &PluginEndpoints{
		Endpoint: lent,
		Lender:   lender,
		PluginProviders: PluginProviders{
			Detection:          detection,
			RowOptions:         rowoptions.NewLists(supervision.Storage, management.Config),
			RunningProviders:   providerbridge.NewRunningSource(management.Activation, management.Activation.ClientLookup()),
			SourceDescriptions: sourcebridge.NewDescriptions(supervision.Storage, management.Config),
		},
	}
	return exec, prefs, endpoints
}

func lendHostEndpoint(management *PluginManagement, processes *PluginSupervision, lender *hostendpoint.Lender) *hostendpoint.Services {
	services := hostendpoint.NewServices(lender.Get)
	toolHost := plugin.NewToolHost(management.Activation, management.Registry)
	manifests := pluginmanifest.Open(management.Registry, management.Registry, processes.Failures)
	revision := getListingRevision(getSearchCatalog(manifests))
	tools := toolbridge.NewCatalog(toolHost, toolHost, processes.Storage, revision, processes.Clock, toolbridge.GetErrorOutput())
	processes.Endpoint.Replace(hostendpoint.NewServer(hostendpoint.Address, hostendpoint.Callers{
		Identity: management.Activation, Leases: management.Activation, Processes: management.Teardown, Files: management.ContentTypes,
		Lent: services,
		Tools: func() hostendpoint.PluginTools {
			return tools
		},
		Health: hostendpoint.HealthCalls{
			Report: pluginhealth.NewReport(processes.Health, management.Registry),
			Clear:  pluginhealth.NewClear(processes.Health),
		},
	}))
	return services
}

func getRuntimeManifests(r PluginRuntime) pluginmanifest.Manifests {
	return pluginmanifest.Open(r.PluginLister, r.PluginResolver, r.PluginFailures)
}
