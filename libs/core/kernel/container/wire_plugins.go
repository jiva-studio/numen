package container

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agenttoolsbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/completion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// StopPlugins stops every plugin the host runs.
func (c Config) StopPlugins(ctx context.Context) error {
	return stopPlugins(plugins(c), ctx)
}

func openPluginCommands(c Config, ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return plugincommand.Run{}, nil, err
	}
	_, work, err := openPluginWork(c, db, v)
	if err != nil {
		return plugincommand.Run{}, nil, errors.Join(err, db.Close())
	}
	return getPluginCommands(plugins(c)).Run, closeAfterWork(ctx, work, db.Close), nil
}

func openedPluginWork(c Config, db *Index, v domain.Vault) (Config, error) {
	opened, _, err := openPluginWork(c, db, v)
	return opened, err
}

func closeAfterWork(ctx context.Context, work *run.PluginWork, closeIndex func() error) func() error {
	return func() error {
		return errors.Join(work.WaitIdle(ctx), closeIndex())
	}
}

func openPluginWork(c Config, db *Index, v domain.Vault) (Config, *run.PluginWork, error) {
	setPluginVaultPath(plugins(c), v.Path)
	versions, err := newTextVersions(vaultOf(c), db)
	if err != nil {
		return c, nil, err
	}
	c.TextVersions = versions
	if err := installCurrentVault(c, db, v); err != nil {
		return c, nil, err
	}
	work := pluginsOpenPluginWork(plugins(c), run.NewTasks(), &run.Lock{})
	setPluginTexts(vaultOf(c))
	return c, work, nil
}

func installCurrentVault(c Config, db *Index, v domain.Vault) error {
	registry, err := c.Registry()
	if err != nil {
		return err
	}
	current := func() (domain.Vault, bool) { return v, true }
	return installPluginVault(plugins(c), registry, db, current, PluginVaultHooks{})
}

// GetPluginServices is the forwarding of the services plugins serve to the windows.
func GetPluginServices(c Config) editor.PluginServices {
	return getPluginServices(plugins(c))
}

// InstallPluginVault lets the plugins work through the host the vault of a list and the index that
// holds them: the one current names, else the one a call names by identity.
func InstallPluginVault(c Config,
	vaults vaultregistry.Ports, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) error {
	return installPluginVault(plugins(c), vaults, db, current, hooks)
}

// PluginRuntime is what runs, finds and addresses the plugins.
type PluginRuntime struct {
	pluginport.PluginActivator
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

// PluginConfiguration is what carries the settings and the secrets to the plugins.
type PluginConfiguration struct {
	pluginport.PluginSettingDeclarer
	pluginport.SecretConfigurator
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
// nothing: the settings then accept every secret, every plugin value and list no plugin. Follow returns the
// subscription to the setting changes that the host follows, or none.
type PluginViews struct {
	Secrets settingsadapter.SecretDeclarations
	Infos   settingsport.PluginInfos
	Checks  settingsport.PluginSettings
	Follow  func(subscribe settingsport.ChangeSubscriber) settingsport.ChangeSubscriber
}

// PluginHost is the plugin host as the capabilities the wiring and the use cases consume, each taken from the
// role of the host that does it. The settings subscription is keyed by the one *PluginHost the container
// made. ApplySetting makes a plugin follow one setting that changed. SetStorage sets the folders the plugins
// store files in and keeps the dependencies in the models folder the host names; RefreshDependencies
// provisions them as the environment names them now.
type PluginHost struct {
	Runtime             PluginRuntime
	Configuration       PluginConfiguration
	Watch               PluginWatch
	Endpoint            *hostendpoint.Services
	ApplySetting        settingsport.PluginSettingApplier
	Views               PluginViews
	SetStorage          func(cacheRoot, dataRoot string)
	RefreshDependencies func()
}

// AdaptPluginHost lends the capabilities of host, a plugin host the caller made, to the container. The
// tests of packages outside the container use it to run a window over a host they hold.
func AdaptPluginHost(host *plugin.Host) *PluginHost {
	return adaptPluginHost(host)
}

func adaptPluginHost(host *plugin.Host) *PluginHost {
	return &PluginHost{
		Runtime: PluginRuntime{
			PluginActivator:    host.Activation,
			PluginRetrier:      host.Retry,
			PluginDiscoverer:   host.Discovery,
			PluginLister:       host.Registry,
			PluginResolver:     host.Registry,
			PluginEventEmitter: host.Events,
			PluginBinding:      host.Binding(),
			ClientLookup:       host.Activation.ClientLookup(),
			TextPlugins:        host.Registry,
			PluginFailures:     host.Failures,
			ConfigWatcher:      host.Config,
		},
		Configuration: PluginConfiguration{
			PluginSettingDeclarer: host.Registry,
			SecretConfigurator:    host.Config,
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
		}).Execute,
		Views: PluginViews{
			Secrets: host.Config, Infos: host.Registry, Checks: host.Settings,
			Follow: func(subscribe settingsport.ChangeSubscriber) settingsport.ChangeSubscriber { return subscribe },
		},
		SetStorage:          func(cacheRoot, dataRoot string) { setStorage(host, cacheRoot, dataRoot) },
		RefreshDependencies: func() { host.Services.SetDependencies(newDependencyRegistry("")) },
	}
}

func lendHostEndpoint(host *plugin.Host) *hostendpoint.Services {
	services := hostendpoint.NewServices()
	host.Services.SetHostEndpoint(hostendpoint.NewServer(hostendpoint.Address, hostendpoint.Callers{
		Identity: host.Activation, Leases: host.Activation,
		Completion: services, Vault: services, Texts: services, Search: services, Window: services, Work: services,
		Dependencies: hostendpoint.DependencySourceFunc(func() hostendpoint.Dependencies {
			if registry := host.Services.GetDependencies(); registry != nil {
				return registry
			}
			return nil
		}),
		Tools: hostendpoint.ToolSourceFunc(func() hostendpoint.PluginTools {
			return agenttoolsbridge.NewCatalog(plugin.NewToolHost(host.Activation, host.Registry), agenttoolsbridge.Full)
		}),
	}))
	return services
}

func setStorage(host *plugin.Host, cacheRoot, dataRoot string) {
	host.Storage.SetStorageRoots(cacheRoot, dataRoot)
	// gomutants:disable-next-line reason="a window started without a models directory gets no dependency registry, which only an installation with downloaded models exercises"
	if models := host.Storage.GetModelsDir(); models != "" {
		// gomutants:disable-next-line reason="the registry is built from the models directory of an installation, which a test does not have"
		host.Services.SetDependencies(newDependencyRegistry(models))
	}
}

// programsLimit is how long the providers have to list the programs they run.
const programsLimit = 8 * time.Second

func pluginsAgentProvider(c pluginsWiring) agentport.AgentProvider {
	return agentbridge.NewRouter(c.RunningHost.Runtime)
}

// programsWait is how long a read of the settings waits for the providers to list their programs.
// Past it the read answers with the programs listed last, so a provider that is slow to start does
// not hold the window back.
const programsWait = time.Second

func getAgentPrograms(c pluginsWiring, provider agentport.AgentProvider, target settingsadapter.Agent) []agentdomain.Program {
	list := func() []agentdomain.Program {
		ctx, stop := context.WithTimeout(context.Background(), programsLimit)
		defer stop()
		programs, err := provider.GetPrograms(ctx, target.GetCommands())
		if err != nil {
			pluginsHandleError(c, fmt.Errorf("list agent programs: %w", err))
		}
		return programs
	}
	return c.Programs.Get(programsWait, list, c.RefreshSettingEvents)
}

func completionAgentSettings(c pluginsWiring) completion.AgentSettings {
	target := c.GetSettingsOrDefaults().Agent
	return completion.AgentSettings{
		IsNamed:  target.IsNamed(),
		Use:      target.GetUse(),
		Commands: target.GetCommands(),
		GetProgram: func(name string) completion.ProgramSettings {
			section := target.GetProgram(name)
			return completion.ProgramSettings{
				Model:   section.Model,
				Command: section.Command,
			}
		},
	}
}
