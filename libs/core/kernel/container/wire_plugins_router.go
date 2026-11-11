package container

import (
	"context"
	"errors"
	"time"

	pluginvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/onnxruntime"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginmanifest"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agenttoolsbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/servicebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	agentcase "github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/completion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

func newCompletionRouter(c pluginsWiring) (*completion.Router, error) {
	provider := pluginsAgentProvider(c)
	programs := completion.NewPrograms(provider, func() completion.AgentSettings { return completionAgentSettings(c) })
	return completion.NewRouter(
		func() completion.Config {
			return completion.Config{Default: completion.Route(settingsGetSettingsOrDefaults(c.settings).AI.Default)}
		},
		programs, agentcase.NewAnswer(programs),
	)
}

func setCompleter(c pluginsWiring) {
	router, err := newCompletionRouter(c)
	if err != nil {
		c.settings.ErrorHandler(err)
		return
	}
	c.RunningHost.Endpoint.SetCompleter(router)
}

func getSettingsAgent(c pluginsWiring) agentport.Agent {
	return completion.NewSettingsAgent(completion.NewPrograms(getToolProvider(c), func() completion.AgentSettings { return completionAgentSettings(c) }))
}

func getToolProvider(c pluginsWiring) agentport.AgentProvider {
	return agentbridge.NewServingRouter(c.RunningHost.Runtime, func(agent agentport.Agent, pluginID string) agentport.Agent {
		return agenttoolsbridge.NewAgent(agent, c.RunningHost.Runtime, agenttoolsbridge.Full, pluginID)
	})
}

func reportDependencies(c pluginsWiring, tasks *run.Tasks) {
	show := run.NewDependencyTasks(tasks)
	c.PluginHost.Watch.ObserveDependencies(func(req manifest.DependencyRequirement, status plugindomain.DependencyStatus) {
		show.Execute(req.ID, status)
	})
}

// pluginInstall is where the plugins are installed from and run: the host that runs them, the catalog
// they are offered from, and the key store their secrets are kept in.
type pluginInstall struct {
	PluginCachePath string
	ServiceDir      string
	PluginHost      *PluginHost
	RunningHost     *PluginHost
	PluginCatalog   catalog.Config
	KeyStore        pluginport.KeyStore
	CredentialsPath func(ctx context.Context) string
}

// pluginsWiring wires the plugin catalog, plugin host, agent provider and dependencies over the settings and
// the vaults the plugins work with.
type pluginsWiring struct {
	pluginInstall
	settings settingsWiring
	vault    vaultWiring
}

func getPluginCatalog(c pluginsWiring) *catalog.Service {
	if c.PluginCatalog.Target == "" {
		return nil
	}
	service, err := catalog.NewService(c.PluginCatalog)
	if err != nil {
		c.settings.ErrorHandler(err)
		return nil
	}
	return service
}

// pluginGracePeriod is how long a plugin has to exit after a stop request.
const pluginGracePeriod = 2 * time.Second

// errSafeMode is an installation started in safe mode, which starts no plugin.
var errSafeMode = errors.New("numen is running in safe mode")

// NewPluginHost creates and initializes a plugin host for the specified target and search directories.
// The host provisions the dependencies the plugins declare; the ONNX Runtime runs on the
// provider NUMEN_ONNX_PROVIDER names (auto when unset). The host is returned also when the discovery of the
// plugins failed, with the failure.
func NewPluginHost(target string, dirs ...string) (*PluginHost, error) {
	host := newPluginRoles(clock.System{}, target, dirs)
	host.ContentTypes.SetContentTypeTable(contenttype.Open())
	host.Services.SetDependencies(newDependencyRegistry(""))
	err := host.Discovery.Discover()
	return adaptPluginHost(host), err
}

func setPluginVaultPath(c pluginsWiring, path string) {
	c.PluginHost.Runtime.SetVaultPath(path)
}

func stopPlugins(c pluginsWiring, ctx context.Context) error {
	return c.PluginHost.Watch.TeardownAll(ctx, pluginGracePeriod)
}

func getPluginCommands(c pluginsWiring) PluginCommandScenarios {
	return newPluginCommands(c, c.PluginHost, nil)
}

// newPluginCommands builds the scenarios over the host; versions are the text versions that a command with
// items lists and runs over, and nil where the window has none.
func newPluginCommands(c pluginsWiring, host *PluginHost, versions plugincommand.VersionItems) PluginCommandScenarios {
	commands, conditions := pluginmanifest.New(host.Runtime), when.NewConditions()
	list := plugincommand.NewList(commands, conditions)
	run := plugincommand.NewRun(commands, commandbridge.Open(host.Runtime), host.Watch, pluginsNewNoteReader(c), getWhenConfig(c), conditions)
	return PluginCommandScenarios{
		List:       list,
		Listing:    plugincommand.NewListCommands(list, host.Watch),
		Run:        run,
		RunCommand: plugincommand.NewRunCommand(list, run, versions),
		ListItems:  plugincommand.NewListItems(list, versions),
	}
}

func getPluginServices(c pluginsWiring) editor.PluginServices {
	return servicebridge.Open(c.PluginHost.Runtime)
}

func getPluginRetrier(c pluginsWiring) func(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
	return c.PluginHost.Runtime.RetryPlugin
}

func reportPluginFailures(c pluginsWiring, tasks *run.Tasks) {
	show := run.NewPluginFailureTasks(tasks)
	c.PluginHost.Runtime.SetOnPluginFailed(show.Report)
	c.PluginHost.Runtime.SetOnFailureCleared(func(pluginID string) {
		show.Clear(pluginID)
		signalPluginChanges(c)
	})
}

func setPluginWork(c pluginsWiring, tasks *run.Tasks, models *run.Lock) {
	pluginsOpenPluginWork(c, tasks, models)
}

func pluginsOpenPluginWork(c pluginsWiring, tasks *run.Tasks, models *run.Lock) *run.PluginWork {
	work := run.NewPluginWork(models, tasks, func(pluginID string) string { return pluginsGetPluginName(c, pluginID) })
	c.PluginHost.Endpoint.SetPluginWork(work)
	return work
}

func pluginsGetPluginName(c pluginsWiring, pluginID string) string {
	if found, isKnown := c.PluginHost.Runtime.GetPlugin(pluginID); isKnown && found.Manifest.Name != "" {
		return found.Manifest.Name
	}
	return pluginID
}

func setPluginVault(c pluginsWiring, v *pluginvault.Access) {
	c.PluginHost.Endpoint.SetPluginVault(v)
}

func setPluginVaultList(c pluginsWiring, list pluginvault.List) {
	c.PluginHost.Endpoint.SetPluginVaultList(list)
}

func openPluginCache(c pluginsWiring) (pluginport.PluginCache, error) {
	retained, err := appstate.OpenPluginCachesIn(c.PluginCachePath)
	if err != nil {
		return nil, err
	}
	return retained, nil
}

func newDependencyRegistry(defaultDir string) pluginport.DependencyRegistry {
	if onnxruntime.IsMockML() {
		return provision.NewRegistry(provision.NewPlaceholderProvisioner(
			provision.GetPlaceholderDir(),
			map[string]func(string) map[string]string{plugindomain.RuntimeONNX: onnxruntime.GetPlaceholderEnv},
		))
	}
	runtimeConfig := onnxruntime.FromEnvironment(clock.System{})
	runtimeConfig.Dir = onnxruntime.GetModelsDir(defaultDir)
	return provision.NewRegistry(
		onnxruntime.NewONNXProvisioner(runtimeConfig),
		provision.NewFileProvisioner(provision.Config{Dir: runtimeConfig.Dir, Offline: onnxruntime.IsOffline(), Clock: clock.System{}}),
	)
}
