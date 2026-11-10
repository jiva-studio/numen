package container

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	pluginvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginmanifest"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/dependencyenv"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agenttoolsbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/servicebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	agentcase "github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/completion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/localsource"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func newCompletionRouter(c pluginsWiring) (*completion.Router, error) {
	provider := pluginsAgentProvider(c)
	if provider == nil {
		return nil, errNoCompletionProvider
	}
	programs := completion.NewPrograms(provider, func() completion.AgentSettings { return completionAgentSettings(c) })
	return completion.NewRouter(
		func() completion.Config {
			return completion.Config{Default: completion.Route(c.GetSettingsOrDefaults().AI.Default)}
		},
		programs, agentcase.NewAnswer(programs),
	)
}

func setCompleter(c pluginsWiring) {
	if c.PluginHost == nil || c.SafeMode {
		return
	}
	router, err := newCompletionRouter(c)
	if err != nil {
		pluginsHandleError(c, err)
		return
	}
	c.PluginHost.Endpoint.SetCompleter(router)
}

func getSettingsAgent(c pluginsWiring) agentport.Agent {
	provider := getToolProvider(c)
	if provider == nil {
		return nil
	}
	return completion.NewSettingsAgent(completion.NewPrograms(provider, func() completion.AgentSettings { return completionAgentSettings(c) }))
}

func getToolProvider(c pluginsWiring) agentport.AgentProvider {
	if c.PluginHost == nil || c.SafeMode {
		return nil
	}
	return agentbridge.NewServingRouter(c.PluginHost.Runtime, func(agent agentport.Agent, pluginID string) agentport.Agent {
		return agenttoolsbridge.NewAgent(agent, c.PluginHost.Runtime, agenttoolsbridge.Full, pluginID)
	})
}

func reportDependencies(c pluginsWiring, tasks *run.Tasks) {
	if c.PluginHost == nil {
		return
	}
	show := run.NewDependencyTasks(tasks)
	c.PluginHost.Watch.ObserveDependencies(func(req manifest.DependencyRequirement, status plugindomain.DependencyStatus) {
		show.Execute(req.ID, status)
	})
}

// pluginsWiring wires the plugin catalog, plugin host, agent provider and dependencies.
type pluginInstall struct {
	PluginCachePath string
	ServiceDir      string
	SafeMode        bool
	PluginHost      *PluginHost
	PluginCatalog   catalog.Config
	PluginDirs      []string
	KeyStore        pluginport.KeyStore
	CredentialsPath func() string
}

type pluginSettingsAccess struct {
	SettingsFiles         *settingsadapter.Files
	SecretGate            *secret.Gate
	Programs              *settingsadapter.ProgramList
	PluginChanges         *settingsusecase.PluginChanges
	ListAppSettings       func() []plugindomain.Setting
	SettingsFile          func() (string, error)
	ReadCheckedSettings   func() (settingsadapter.File, error)
	GetSettingsOrDefaults func() settingsadapter.File
	RefreshSettingEvents  func()
}

type pluginVaultAccess struct {
	TextVersions *textversion.Versions
	VaultReaders vaultport.VaultReaders
	NoteWriters  vaultport.NoteWriters
	VaultLocks   vaultport.VaultLocks
	VaultWatcher vaultport.VaultWatcher
	MakeRefresh  func(*vault.Index, noteport.NoteRepository) vault.Refresh
	BuildNotes   func(*vault.Index, edit.IndexUpdater) note.Scenarios
	Moves        func(context.Context) (<-chan struct{}, error)
}

type pluginsWiring struct {
	pluginInstall
	pluginSettingsAccess
	pluginVaultAccess
	ErrorHandler func(error)
}

func getPluginCatalog(c pluginsWiring) *catalog.Service {
	if c.PluginCatalog.Target == "" || c.PluginHost == nil {
		return nil
	}
	config := c.PluginCatalog
	config.Sources = append(
		[]catalog.Source{localsource.New(slices.Clone(c.PluginDirs), localsource.SettingsFunc(c.PluginHost.Runtime.IsPluginEnabled))},
		config.Sources...)
	service, err := catalog.NewService(config)
	if err != nil {
		pluginsHandleError(c, err)
		return nil
	}
	return service
}

func pluginsHandleError(c pluginsWiring, err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
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
	host := plugin.NewHost(clock.System{}, target, dirs...)
	host.ContentTypes.SetContentTypeTable(sharedmimeinfo.Open())
	host.Services.SetDependencies(dependencyenv.Open(""))
	err := host.Discovery.Discover()
	return adaptPluginHost(host), err
}

func pluginsResolvePlugin(c pluginsWiring, capability string) (*PluginHost, error) {
	if c.PluginHost == nil {
		return nil, &plugindomain.UnavailableError{Capability: capability, Reason: plugindomain.NotInstalled}
	}
	if c.SafeMode {
		return nil, &plugindomain.UnavailableError{Capability: capability, Reason: plugindomain.Disabled, Err: errSafeMode}
	}
	if _, err := c.PluginHost.Runtime.ResolveCapability(capability); err != nil {
		return nil, err
	}
	return c.PluginHost, nil
}

func setSafeMode(c pluginsWiring, enabled bool) pluginsWiring {
	c.SafeMode = enabled
	if c.PluginHost != nil {
		c.PluginHost.Runtime.SetSafeMode(enabled)
	}
	return c
}

func setPluginVaultPath(c pluginsWiring, path string) {
	if c.PluginHost != nil {
		c.PluginHost.Runtime.SetVaultPath(path)
	}
}

func stopPlugins(c pluginsWiring, ctx context.Context) error {
	if c.PluginHost == nil {
		return nil
	}
	return c.PluginHost.Watch.TeardownAll(ctx, pluginGracePeriod)
}

func getPluginCommands(c pluginsWiring) plugincommand.Commands {
	if c.PluginHost == nil {
		return plugincommand.NewCommands(nil, nil, nil, nil, nil)
	}
	return plugincommand.NewCommands(pluginmanifest.New(c.PluginHost.Runtime), commandbridge.Open(c.PluginHost.Runtime), c.PluginHost.Watch, pluginsNewNoteReader(c), getWhenConfig(c))
}

// getPluginServices is where a window forwards the calls it receives for the services plugins serve.
func getPluginServices(c pluginsWiring) shell.PluginServices {
	if c.PluginHost == nil {
		return servicebridge.NoServices{}
	}
	return servicebridge.Open(c.PluginHost.Runtime)
}

func getPluginRetrier(c pluginsWiring) func(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
	return func(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
		if c.PluginHost == nil {
			return nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.NotInstalled}
		}
		return c.PluginHost.Runtime.RetryPlugin(ctx, pluginID)
	}
}

func reportPluginFailures(c pluginsWiring, tasks *run.Tasks) {
	if c.PluginHost == nil {
		return
	}
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
	if c.PluginHost != nil {
		c.PluginHost.Endpoint.SetPluginWork(work)
	}
	return work
}

func pluginsGetPluginName(c pluginsWiring, pluginID string) string {
	if c.PluginHost == nil {
		return pluginID
	}
	if found, isKnown := c.PluginHost.Runtime.GetPlugin(pluginID); isKnown && found.Manifest.Name != "" {
		return found.Manifest.Name
	}
	return pluginID
}

func setPluginVault(c pluginsWiring, v *pluginvault.Access) {
	if c.PluginHost != nil {
		c.PluginHost.Endpoint.SetPluginVault(v)
	}
}

func setPluginVaultList(c pluginsWiring, list pluginvault.List) {
	if c.PluginHost != nil {
		c.PluginHost.Endpoint.SetPluginVaultList(list)
	}
}

func openPluginCache(c pluginsWiring) (pluginport.PluginCache, error) {
	retained, err := appstate.OpenPluginCachesIn(c.PluginCachePath)
	if err != nil {
		return nil, err
	}
	return retained, nil
}
