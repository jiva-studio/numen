package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/servicebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/toolbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/license"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugindependency"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginmanifest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginprocess"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func installCurrentVault(install pluginInstall, vaultDeps vaultParts, db *vaultIndex, v domain.Vault) error {
	registry, err := openVaultRegistry(vaultDeps.RegistryPath)
	if err != nil {
		return err
	}
	current := func() (domain.Vault, bool) { return v, true }
	return installPluginVault(install, vaultDeps, registry, db, current, pluginVaultHooks{})
}

type installedPlugins interface {
	plugin.PluginEnablement
	textversion.TextPlugins
	pluginmanifest.PluginLister
	pluginport.PluginManifests
	ListPlugins() []manifest.DiscoveredPlugin
}

// pluginResolver supplies the plugin lookups used during host assembly.
type pluginResolver interface {
	servicebridge.ServiceResolver
	sourcebridge.CapabilityResolver
	pluginmanifest.PluginLookup
	ResolveNoteProfile(noteType domain.NoteType) (domain.NoteProfile, bool)
}

// pluginRuntime is what runs, finds and addresses the plugins.
type pluginRuntime struct {
	bridge.Activator
	pluginRetrier
	Listing   installedPlugins
	Resolver  pluginResolver
	Manifests pluginmanifest.Manifests
	pluginEventEmitter
	pluginBinding
	transport.ClientLookup
	pluginmanifest.PluginFailures
	sourcebridge.ConfigWatcher
}

// pluginConfiguration is what carries the settings to the plugins.
type pluginConfiguration struct {
	pluginSettingDeclarer
	configRefresher
	settingsusecase.PluginConfig
	settingsusecase.PluginSettings
}

// PluginWatch is what ends the plugins, reports their dependencies and names the content types they give.
type PluginWatch struct {
	pluginShutdown
	dependencyReporter
	contenttypeport.ContentTypeTable
}

// pluginViews are the parts of the host that the settings take as they are, and that stay unset where the host runs
// nothing: the settings then accept every plugin value and list no plugin, and no connection type exists. Follow returns the
// subscription to the setting changes that the host follows, or none.
type pluginViews struct {
	Types        connectionport.Types
	Operations   connection.OperationTitles
	Declarations license.Declarations
	Infos        settingsusecase.PluginInfos
	Checks       settingsusecase.PluginSettings
	Follow       func(subscribe settingsusecase.ChangeSubscriber) settingsusecase.ChangeSubscriber
}

// PluginExecution runs plugins and tracks their processes.
type PluginExecution struct {
	Runtime   pluginRuntime
	Watch     PluginWatch
	Health    *pluginhealth.Items
	Processes pluginprocess.Processes
}

// PluginPreferences applies settings and storage locations to the installation's plugins.
type PluginPreferences struct {
	Configuration       pluginConfiguration
	Views               pluginViews
	ApplySetting        settingsusecase.PluginSettingApplier
	SetStorage          func(cacheRoot, dataRoot string)
	RefreshDependencies func()
	Detection           connectionport.DetectedConnections
}

// pluginProviders connects plugin providers to search and model clients.
type pluginProviders struct {
	Detection          *providerbridge.Detection
	RowOptions         *connection.Lists
	RunningProviders   *providerbridge.RunningSource
	SourceDescriptions *sourcebridge.Descriptions
}

// PluginEndpoints connects the host RPC services and the providers served by plugins.
type PluginEndpoints struct {
	workLock run.Lock
	Endpoint *hostendpoint.Services
	Lender   *hostendpoint.Lender
	pluginProviders
}

// CreatePluginHost connects the process and management roles to their consumers.
func CreatePluginHost(management *PluginManagement, roles *PluginSupervision) (*PluginExecution, *PluginPreferences, *PluginEndpoints) {
	lender := hostendpoint.NewLender()
	lent := lendHostEndpoint(management, roles, lender)
	types := plugin.NewConnectionTypes(management.Registry)
	detection := providerbridge.NewDetection(types, providerbridge.NewQuestionSource(management.Activation), roles.Clock)
	management.Activation.SetOnStart(detection.DetectInBackground)
	exec := &PluginExecution{
		Runtime: pluginRuntime{
			Activator:          management.Activation,
			pluginRetrier:      management.Retry,
			Listing:            management.Registry,
			Resolver:           management.Registry,
			Manifests:          pluginmanifest.Open(management.Registry, management.Registry, roles.Failures),
			pluginEventEmitter: management.Events,
			pluginBinding:      plugin.NewBinding(management.Teardown, management.Mode),
			ClientLookup:       management.Activation.ClientLookup(),
			PluginFailures:     roles.Failures,
			ConfigWatcher:      management.Config,
		},
		Watch: PluginWatch{
			pluginShutdown:     management.Teardown,
			dependencyReporter: roles.Services,
			ContentTypeTable:   management.ContentTypes,
		},
		Health:    roles.Health,
		Processes: roles.RunningProcesses,
	}
	prefs := &PluginPreferences{
		Configuration: pluginConfiguration{
			pluginSettingDeclarer: management.Registry,
			configRefresher:       management.Config,
			PluginConfig:          management.Settings,
			PluginSettings:        management.Settings,
		},
		ApplySetting: settingsusecase.NewApplyPluginSetting(settingsusecase.ApplyPluginSettingDeps{
			Values: management.Settings, Stopper: management.Teardown, Failures: roles.Failures, Listeners: management.Listeners, Push: management.Config,
			Health: roles.Health,
		}).Execute,
		Views: pluginViews{
			Types: types, Operations: types,
			Declarations: management.Registry, Infos: management.Registry, Checks: management.Settings,
			Follow: func(subscribe settingsusecase.ChangeSubscriber) settingsusecase.ChangeSubscriber { return subscribe },
		},
		SetStorage: func(cacheRoot, dataRoot string) {
			setPluginStorage(roles, cacheRoot, dataRoot, func(modelsDir string) supervision.Dependencies {
				return newDependencyRegistry(roles.Clock, roles.Services.Observe, modelsDir)
			})
			detection.SetFiles(roles.Storage)
		},
		RefreshDependencies: func() {
			roles.Dependencies.Replace(newDependencyRegistry(roles.Clock, roles.Services.Observe, ""))
		},
		Detection: detection,
	}
	endpoints := &PluginEndpoints{
		Endpoint: lent,
		Lender:   lender,
		pluginProviders: pluginProviders{
			Detection:          detection,
			RowOptions:         connection.NewLists(roles.Storage, management.Config),
			RunningProviders:   providerbridge.NewRunningSource(management.Activation, management.Activation.ClientLookup()),
			SourceDescriptions: sourcebridge.NewDescriptions(roles.Storage, management.Config),
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
		Dependencies: plugindependency.NewFetch(getDependencyProvisioner(processes.Services)),
		Identity:     management.Activation, Leases: management.Activation, Processes: management.Teardown, Files: management.ContentTypes,
		Lent: services, CallCancels: searchcancel.NewSearches(processes.Clock),
		GetTools: func() hostendpoint.PluginTools {
			return tools
		},
		Health: hostendpoint.HealthCalls{
			Report: pluginhealth.NewReport(processes.Health, management.Registry),
			Clear:  pluginhealth.NewClear(processes.Health),
		},
	}))
	return services
}
