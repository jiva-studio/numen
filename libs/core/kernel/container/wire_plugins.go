package container

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/onnxruntime"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agenttoolsbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/manifestlist"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/servicebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/textbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	agentcase "github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/completion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/trigger"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/localsource"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ReportDependencies keeps one task in the list for each dependency being provisioned.
func (c Config) ReportDependencies(tasks *run.Tasks) {
	reportDependencies(c.plugins(), tasks)
}

// GetPluginCatalog is the catalog of the plugins on disk, the disabled ones too, and of the sources
// the configuration names.
func (c Config) GetPluginCatalog() *catalog.Service {
	return getPluginCatalog(c.plugins())
}

func (c Config) resolvePlugin(capability string) (*PluginHost, error) {
	return pluginsResolvePlugin(c.plugins(), capability)
}

// SetSafeMode is the configuration started in safe mode or out of it, and its plugin host with it.
func (c Config) SetSafeMode(enabled bool) Config {
	c.SafeMode = setSafeMode(c.plugins(), enabled).SafeMode
	return c
}

// SetPluginVaultPath is the vault the plugins' paths are inside.
func (c Config) SetPluginVaultPath(path string) {
	setPluginVaultPath(c.plugins(), path)
}

// StopPlugins stops every plugin the host runs.
func (c Config) StopPlugins(ctx context.Context) error {
	return stopPlugins(c.plugins(), ctx)
}

// GetPluginCommands is the use cases over the commands the plugins declare.
func (c Config) GetPluginCommands() plugincommand.Commands {
	return getPluginCommands(c.plugins())
}

// OpenPluginCommands opens the index of the vault and lets the plugins work it, its text versions
// and the queue through the host with no window.
func (c Config) OpenPluginCommands(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return plugincommand.Run{}, nil, err
	}
	_, work, err := c.openPluginWork(db, v)
	if err != nil {
		return plugincommand.Run{}, nil, errors.Join(err, db.Close())
	}
	return c.GetPluginCommands().Run, closeAfterWork(ctx, work, db.Close), nil
}

// OpenPluginWork lets the plugins work one vault, its text versions and the queue through the host
// with no window, over the index given.
func (c Config) OpenPluginWork(ctx context.Context, db *vault.Index, v domain.Vault) (Config, error) {
	opened, _, err := c.openPluginWork(db, v)
	return opened, err
}

func closeAfterWork(ctx context.Context, work *run.PluginWork, closeIndex func() error) func() error {
	return func() error {
		return errors.Join(work.WaitIdle(ctx), closeIndex())
	}
}

func (c Config) openPluginWork(db *vault.Index, v domain.Vault) (Config, *run.PluginWork, error) {
	c.SetPluginVaultPath(v.Path)
	versions, err := c.NewTextVersions(db)
	if err != nil {
		return c, nil, err
	}
	c.TextVersions = versions
	registry, err := c.Registry()
	if err != nil {
		return c, nil, err
	}
	current := func() (domain.Vault, bool) { return v, true }
	if err := c.InstallPluginVault(registry, db, current, PluginVaultHooks{}); err != nil {
		return c, nil, err
	}
	work := pluginsOpenPluginWork(c.plugins(), run.NewTasks(), &run.Lock{})
	c.SetPluginTexts()
	return c, work, nil
}

// GetPluginServices is the forwarding of the services plugins serve to the windows.
func (c Config) GetPluginServices() PluginServices {
	return getPluginServices(c.plugins())
}

// GetPluginRetrier restarts, on request from the window, a plugin the host has suspended.
func (c Config) GetPluginRetrier() func(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
	return getPluginRetrier(c.plugins())
}

// ReportPluginFailures keeps one task in the list for each plugin the host suspends.
func (c Config) ReportPluginFailures(tasks *run.Tasks) {
	reportPluginFailures(c.plugins(), tasks)
}

// SetPluginWork opens the lock the models are held under and the list of tasks to the plugins.
func (c Config) SetPluginWork(tasks *run.Tasks, models *run.Lock) {
	setPluginWork(c.plugins(), tasks, models)
}

func (c Config) getPluginName(pluginID string) string {
	return pluginsGetPluginName(c.plugins(), pluginID)
}

// SetPluginVaultList sets the list of vaults exposed to plugins.
func (c Config) SetPluginVaultList(list hostendpoint.VaultList) {
	setPluginVaultList(c.plugins(), list)
}

// OpenPluginCache is where a plugin keeps what it worked out for a vault.
func (c Config) OpenPluginCache() (pluginport.PluginCache, error) {
	return openPluginCache(c.plugins())
}

// GetPluginDir is the folder of an installed plugin, and false for one not installed.
func (c Config) GetPluginDir(id string) (string, bool) {
	return getPluginDir(c.plugins(), id)
}

// GetFrontendPluginIDs are the plugins of the host's target that declare a frontend, in the order of
// their ids.
func (c Config) GetFrontendPluginIDs() []string {
	return getFrontendPluginIDs(c.plugins())
}

// SetPluginWindow sets the window exposed to plugins.
func (c Config) SetPluginWindow(window hostendpoint.WindowControl) {
	setPluginWindow(c.plugins(), window)
}

// OpenSecrets returns the use cases on the secrets shared between plugins.
func (c Config) OpenSecrets() *secret.Secrets {
	return openSecrets(c.plugins())
}

// AgentProvider runs tasks on the programs the plugins with the agent provider capability run.
func (c Config) AgentProvider() agentport.AgentProvider {
	return pluginsAgentProvider(c.plugins())
}

// GetSettingsAgent runs a window's tasks on the programs the plugins run, under the settings read at
// every task.
func (c Config) GetSettingsAgent() agentport.Agent {
	return getSettingsAgent(c.plugins())
}

// OpenChooseTabKind is the choice of the kind of tab a note opens in, over the tab kinds of the
// enabled plugins.
func (c Config) OpenChooseTabKind() *tabkind.Choose {
	return openChooseTabKind(c.plugins())
}

// WatchPluginChanges reports each time the plugin host applied a batch of settings, which enables,
// disables and configures plugins, and each time it forgot the failure of a plugin, as after a retry
// that succeeded. The returned function stops the reports.
func (c Config) WatchPluginChanges() (<-chan struct{}, func()) {
	return watchPluginChanges(c.plugins())
}

// InstallPluginVault lets the plugins work through the host the vault of a list and the index that
// holds them: the one current names, else the one a call names by identity.
func (c Config) InstallPluginVault(
	vaults vaultregistry.Ports, db *vault.Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) error {
	return installPluginVault(c.plugins(), vaults, db, current, hooks)
}

// PluginHost is the plugin host as the capabilities the wiring and the use cases consume, each taken from
// the role of the host that does it. The settings subscription is keyed by the one *PluginHost the container
// made. ApplySetting makes a plugin follow one setting that changed. SetStorage sets the folders the plugins
// store files in and keeps the dependencies in the models folder the host names; RefreshDependencies
// provisions them as the environment names them now.
type PluginHost struct {
	pluginport.PluginActivator
	pluginport.PluginRetrier
	pluginport.PluginDiscoverer
	pluginport.PluginLister
	pluginport.PluginResolver
	pluginport.PluginSettingDeclarer
	pluginport.PluginEventEmitter
	pluginport.SecretConfigurator
	pluginport.ConfigWatcher
	pluginport.SettingsPush
	pluginport.SettingChecker
	pluginport.PersistHook
	pluginport.PluginFailures
	pluginport.PluginShutdown
	pluginport.PluginBinding
	pluginport.DependencyReporter
	contenttypeport.ContentTypeTable
	*hostendpoint.Services
	ApplySetting        settingsport.PluginSettingApplier
	SetStorage          func(cacheRoot, dataRoot string)
	RefreshDependencies func()
}

// LendPluginHost lends the capabilities of host, a plugin host the caller made, to the container. The
// tests of packages outside the container use it to run a window over a host they hold.
func LendPluginHost(host *plugin.Host) *PluginHost {
	return lendPluginHost(host)
}

func lendPluginHost(host *plugin.Host) *PluginHost {
	return &PluginHost{
		Services:              lendHostEndpoint(host),
		PluginActivator:       host.Activation,
		PluginRetrier:         host.Activation,
		PluginDiscoverer:      host.Discovery,
		PluginLister:          host.Registry,
		PluginResolver:        host.Registry,
		PluginSettingDeclarer: host.Registry,
		PluginEventEmitter:    host.Events,
		SecretConfigurator:    host.Config,
		ConfigWatcher:         host.Config,
		SettingsPush:          host.Settings,
		SettingChecker:        host.Settings,
		PersistHook:           host.Settings,
		PluginFailures:        host.Failures,
		PluginShutdown:        host.Teardown,
		PluginBinding:         host.Mode,
		DependencyReporter:    host.Services,
		ContentTypeTable:      host.ContentTypes,
		ApplySetting: settingsusecase.NewApplyPluginSetting(settingsusecase.ApplyPluginSettingDeps{
			Values: host.Settings, Stopper: host.Teardown, Failures: host.Failures, Listeners: host.Listeners, Push: host.Config,
		}).Execute,
		SetStorage:          func(cacheRoot, dataRoot string) { setStorage(host, cacheRoot, dataRoot) },
		RefreshDependencies: func() { host.Services.SetDependencies(GetDependencyRegistry("")) },
	}
}

// lendHostEndpoint makes the endpoint the plugins call the host through, and gives it to the host. The
// returned services are what the endpoint lends.
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
		host.Services.SetDependencies(GetDependencyRegistry(models))
	}
}

// programsLimit is how long the providers have to list the programs they run.
const programsLimit = 8 * time.Second

func pluginsAgentProvider(c pluginsWiring) agentport.AgentProvider {
	if c.PluginHost == nil || c.SafeMode {
		return nil
	}
	return agentbridge.NewRouter(c.PluginHost)
}

// programsWait is how long a read of the settings waits for the providers to list their programs.
// Past it the read answers with the programs listed last, so a provider that is slow to start does
// not hold the window back.
const programsWait = time.Second

func (c Config) getCredentialsPath() string { return getCredentialsPath(c.plugins()) }

func (c Config) setSecretSource() { setSecretSource(c.plugins()) }

func (c Config) setCompleter() { setCompleter(c.plugins()) }

func (c Config) getAgentPrograms(provider agentport.AgentProvider, target settingsadapter.Agent) []agentdomain.Program {
	return getAgentPrograms(c.plugins(), provider, target)
}

func getAgentPrograms(c pluginsWiring, provider agentport.AgentProvider, target settingsadapter.Agent) []agentdomain.Program {
	if provider == nil {
		return nil
	}
	list := func() []agentdomain.Program {
		ctx, stop := context.WithTimeout(context.Background(), programsLimit)
		defer stop()
		programs, err := provider.GetPrograms(ctx, target.GetCommands())
		if err != nil {
			pluginsHandleError(c, fmt.Errorf("list agent programs: %w", err))
		}
		return programs
	}
	if c.Programs == nil {
		return list()
	}
	return c.Programs.Get(programsWait, list, c.RefreshSettingEvents)
}

var errNoCompletionProvider = errors.New("no plugin host runs agent programs")

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
	c.PluginHost.SetCompleter(router)
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
	return agentbridge.NewServingRouter(c.PluginHost, func(agent agentport.Agent, pluginID string) agentport.Agent {
		return agenttoolsbridge.NewAgent(agent, c.PluginHost, agenttoolsbridge.Full, pluginID)
	})
}

const dependencyTaskPrefix = "provision:"

// envMockML names the variable that runs the plugins on mock models.
const envMockML = "NUMEN_MOCK_ML"

// GetDependencyRegistry returns the registry of the dependencies the plugins provision, in the folder
// given unless the environment names another.
func GetDependencyRegistry(defaultDir string) pluginport.DependencyRegistry {
	if os.Getenv(envMockML) == "true" {
		return provision.NewRegistry(newPlaceholders())
	}
	dir := defaultDir
	if models := os.Getenv(onnxruntime.EnvDir); models != "" {
		dir = models
	}
	runtimeConfig := onnxruntime.FromEnvironment(clock.System{})
	runtimeConfig.Dir = dir
	return provision.NewRegistry(
		onnxruntime.NewONNXProvisioner(runtimeConfig),
		provision.NewFileProvisioner(provision.Config{Dir: dir, Offline: onnxruntime.IsOffline(), Clock: clock.System{}}),
	)
}

func newPlaceholders() *provision.PlaceholderProvisioner {
	dir, err := os.UserCacheDir()
	if err == nil {
		dir = filepath.Join(dir, "numen", "placeholder-dependencies")
	} else if dir, err = os.MkdirTemp("", "numen-placeholder-dependencies-"); err != nil {
		// Placeholders written under the working folder still let a mock run start.
		dir = "numen-placeholder-dependencies"
	}
	return provision.NewPlaceholderProvisioner(dir, map[string]func(string) map[string]string{
		plugindomain.RuntimeONNX: func(primary string) map[string]string {
			return map[string]string{onnxruntime.EnvLibrary: primary, onnxruntime.EnvProvider: "cpu"}
		},
	})
}

func reportDependencies(c pluginsWiring, tasks *run.Tasks) {
	if c.PluginHost == nil {
		return
	}
	c.PluginHost.ObserveDependencies(func(req manifest.DependencyRequirement, status plugindomain.DependencyStatus) {
		id := dependencyTaskPrefix + req.ID
		switch status.State {
		case plugindomain.DependencyStateReady:
			tasks.Remove(id)
		case plugindomain.DependencyStateMissing, plugindomain.DependencyStateProvisioning, plugindomain.DependencyStateFailed:
			tasks.Set(work.Task{
				ID: id, Title: "Downloading " + req.ID, Unit: work.Bytes,
				Count: status.BytesDownloaded, Total: status.BytesTotal, Error: status.Error, IsUserStarted: false,
			})
		}
	})
}

// pluginsWiring wires the plugin catalog, plugin host, agent provider and dependencies.
type pluginsWiring struct {
	PluginCachePath       string
	ServiceDir            string
	SafeMode              bool
	PluginHost            *PluginHost
	PluginCatalog         catalog.Config
	PluginDirs            []string
	KeyStore              pluginport.KeyStore
	SettingsFiles         *settingsadapter.Files
	SecretGate            *secret.Gate
	Programs              *settingsadapter.ProgramList
	VaultReaders          vaultport.VaultReaders
	NoteWriters           vaultport.NoteWriters
	VaultLocks            vaultport.VaultLocks
	VaultWatcher          vaultport.VaultWatcher
	ErrorHandler          func(error)
	SettingsFile          func() (string, error)
	ReadCheckedSettings   func() (settingsadapter.File, error)
	GetSettingsOrDefaults func() settingsadapter.File
	RefreshSettingEvents  func()
	MakeRefresh           func(*vault.Index, noteport.NoteRepository) vault.Refresh
	BuildNotes            func(*vault.Index, edit.IndexUpdater) note.Scenarios
	Moves                 func(context.Context) (<-chan struct{}, error)
	PluginChanges         *settingsusecase.PluginChanges
	TextVersions          *textversion.Versions
	ListAppSettings       func() []plugindomain.Setting
	CredentialsPath       func() string
}

func getPluginCatalog(c pluginsWiring) *catalog.Service {
	if c.PluginCatalog.Target == "" || c.PluginHost == nil {
		return nil
	}
	config := c.PluginCatalog
	config.Sources = append(
		[]catalog.Source{localsource.New(slices.Clone(c.PluginDirs), localsource.SettingsFunc(c.PluginHost.IsPluginEnabled))},
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

const pluginFailureTaskPrefix = "plugin-failed:"

// NewPluginHost creates and initializes a plugin host for the specified target and search directories.
// The host provisions the dependencies the plugins declare; the ONNX Runtime runs on the
// provider NUMEN_ONNX_PROVIDER names (auto when unset).
func NewPluginHost(target string, dirs ...string) *PluginHost {
	return lendPluginHost(openPluginHost(target, dirs...))
}

func openPluginHost(target string, dirs ...string) *plugin.Host {
	host := plugin.NewHost(clock.System{}, target, dirs...)
	host.ContentTypes.SetContentTypeTable(sharedmimeinfo.Open())
	host.Services.SetDependencies(GetDependencyRegistry(""))
	if err := host.Discovery.Discover(); err != nil {
		fmt.Fprintf(os.Stderr, "numen: discover plugins: %v\n", err)
	}
	return host
}

func pluginsResolvePlugin(c pluginsWiring, capability string) (*PluginHost, error) {
	if c.PluginHost == nil {
		return nil, &plugindomain.UnavailableError{Capability: capability, Reason: plugindomain.NotInstalled}
	}
	if c.SafeMode {
		return nil, &plugindomain.UnavailableError{Capability: capability, Reason: plugindomain.Disabled, Err: errSafeMode}
	}
	if _, err := c.PluginHost.ResolveCapability(capability); err != nil {
		return nil, err
	}
	return c.PluginHost, nil
}

func setSafeMode(c pluginsWiring, enabled bool) pluginsWiring {
	c.SafeMode = enabled
	if c.PluginHost != nil {
		c.PluginHost.SetSafeMode(enabled)
	}
	return c
}

func setPluginVaultPath(c pluginsWiring, path string) {
	if c.PluginHost != nil {
		c.PluginHost.SetVaultPath(path)
	}
}

func stopPlugins(c pluginsWiring, ctx context.Context) error {
	if c.PluginHost == nil {
		return nil
	}
	return c.PluginHost.TeardownAll(ctx, pluginGracePeriod)
}

func getPluginCommands(c pluginsWiring) plugincommand.Commands {
	if c.PluginHost == nil {
		return plugincommand.NewCommands(nil, nil, nil, nil, nil)
	}
	return plugincommand.NewCommands(manifestlist.Open(c.PluginHost), commandbridge.Open(c.PluginHost), c.PluginHost, pluginsNewNoteReader(c), getWhenConfig(c))
}

// PluginServices is where a window forwards the calls it receives for the services plugins serve.
type PluginServices = interface {
	http.Handler
	// Serves reports whether an enabled plugin serves the service the path names.
	Serves(urlPath string) bool
	// ServeRoute forwards a request on the plugin route, /plugin/<id>/..., to that plugin.
	ServeRoute(w http.ResponseWriter, r *http.Request)
	// ListFrameOrigins lists the origins the pages of every enabled plugin frame.
	ListFrameOrigins() []string
	// GetFrameOrigins lists the origins the pages of one enabled plugin frame.
	GetFrameOrigins(pluginID string) []string
}

func getPluginServices(c pluginsWiring) PluginServices {
	if c.PluginHost == nil {
		return servicebridge.NoServices{}
	}
	return servicebridge.Open(c.PluginHost)
}

func getPluginRetrier(c pluginsWiring) func(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
	return func(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
		if c.PluginHost == nil {
			return nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.NotInstalled}
		}
		return c.PluginHost.RetryPlugin(ctx, pluginID)
	}
}

func reportPluginFailures(c pluginsWiring, tasks *run.Tasks) {
	if c.PluginHost == nil {
		return
	}
	c.PluginHost.SetOnPluginFailed(func(failure plugindomain.Failure) {
		tasks.Set(work.Task{
			ID:         pluginFailureTaskPrefix + failure.PluginID,
			Title:      "Plugin stopped",
			About:      failure.Name,
			Error:      failure.Reason,
			PluginID:   failure.PluginID,
			PluginName: failure.Name,
		})
	})
	c.PluginHost.SetOnFailureCleared(func(pluginID string) {
		tasks.Remove(pluginFailureTaskPrefix + pluginID)
		signalPluginChanges(c)
	})
}

func setPluginWork(c pluginsWiring, tasks *run.Tasks, models *run.Lock) {
	pluginsOpenPluginWork(c, tasks, models)
}

func pluginsOpenPluginWork(c pluginsWiring, tasks *run.Tasks, models *run.Lock) *run.PluginWork {
	work := run.NewPluginWork(models, tasks, func(pluginID string) string { return pluginsGetPluginName(c, pluginID) })
	if c.PluginHost != nil {
		c.PluginHost.SetPluginWork(work)
	}
	return work
}

func pluginsGetPluginName(c pluginsWiring, pluginID string) string {
	if c.PluginHost == nil {
		return pluginID
	}
	if found, isKnown := c.PluginHost.GetPlugin(pluginID); isKnown && found.Manifest.Name != "" {
		return found.Manifest.Name
	}
	return pluginID
}

func setPluginVault(c pluginsWiring, v *hostendpoint.VaultAccess) {
	if c.PluginHost != nil {
		c.PluginHost.SetPluginVault(v)
	}
}

func setPluginVaultList(c pluginsWiring, list hostendpoint.VaultList) {
	if c.PluginHost != nil {
		c.PluginHost.SetPluginVaultList(list)
	}
}

func openPluginCache(c pluginsWiring) (pluginport.PluginCache, error) {
	if c.PluginCachePath != "" {
		return appstate.PluginCachesAt(filepath.Join(c.PluginCachePath, "plugins")), nil
	}
	retained, err := appstate.OpenPluginCaches()
	if err != nil {
		return nil, err
	}
	return retained, nil
}

// PluginVaultHooks is what a window adds to the vault it lets the plugins work.
type PluginVaultHooks struct {
	Written    func()
	IndexState func(ctx context.Context, v domain.Vault) (plugindomain.IndexState, error)
	Moved      func(ctx context.Context) (<-chan struct{}, error)
	Notes      note.Scenarios
}

func hasReaders(notes note.Scenarios) bool {
	return notes.Place.Move.Readers() != nil
}

func getNoteChanges(notes note.Scenarios) hostvault.NoteChanges {
	return hostvault.NoteChanges{
		Rename: notes.Place.Rename, Move: notes.Place.Move, Remove: notes.Place.RemoveNote, Links: notes.Edit.Linking, AddLinks: notes.Refer.AddLinks,
	}
}

func openNoteWriter(c pluginsWiring, v domain.Vault) (hostvault.NoteWriter, error) {
	writer, err := c.NoteWriters.OpenNoteWriter(v)
	if err != nil {
		return nil, err
	}
	return writer, nil
}

func lockVault(c pluginsWiring, ctx context.Context, v domain.Vault) (func(), error) {
	return c.VaultLocks.Lock(ctx, v)
}

func installPluginVault(c pluginsWiring,
	registry vaultregistry.Ports, db *vault.Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) error {
	opened, err := pluginsOpenPluginVault(c, registry, db, current, hooks)
	if err != nil {
		return err
	}
	setPluginVault(c, opened)
	return nil
}

func pluginsOpenPluginVault(c pluginsWiring,
	registry vaultregistry.Ports, db *vault.Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) (*hostendpoint.VaultAccess, error) {
	cache, err := openPluginCache(c)
	if err != nil {
		return nil, err
	}
	refresh := c.MakeRefresh(db, db.Notes())
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	scenarios := c.BuildNotes(db, index)
	connections := hostvault.Connections{
		Reader:  note.NewConnectionReader(scenarios),
		Writer:  note.NewConnectionWriter(scenarios),
		Joiner:  note.NewConnectionJoiner(scenarios),
		Creator: note.NewRelatedNoteCreator(scenarios),
	}
	arranger := hostvault.Arranger{
		Notes:  getNoteChanges(scenarios),
		Checks: getProblemChecks(vault.NewStandardGetProblems(db.Problems())),
	}
	if hasReaders(hooks.Notes) {
		arranger.Window, arranger.WindowVault = getNoteChanges(hooks.Notes), current
	}
	stores := derivedfile.PluginStores{Options: vaultconfig.Options{ServiceDir: c.ServiceDir}}
	vaults := hostvault.Vault{
		Current:     current,
		Stat:        file.NewStatFile(c.VaultReaders),
		ReadFile:    file.NewReadFile(c.VaultReaders),
		RangeReader: file.Read{Readers: c.VaultReaders},
		OpenWriter:  func(v domain.Vault) (hostvault.NoteWriter, error) { return openNoteWriter(c, v) },
		LockVault:   func(ctx context.Context, v domain.Vault) (func(), error) { return lockVault(c, ctx, v) },
		Notes:       db.NoteRefs(),
		Types:       db.NoteTypes(),
		NamedPaths:  db.NamedPaths(),
		TypedPaths:  db.TypedPaths(),
		Links:       db.Links(),
		Connections: connections,
		Problems:    db.Problems(),
		Arranger:    arranger,
		StoreReaders: func(v domain.Vault, pluginID string) (hostendpoint.StoreReader, error) {
			return stores.Open(v, pluginID)
		},
		StoreWriters: func(v domain.Vault, pluginID string) (hostendpoint.StoreWriter, error) {
			return stores.OpenWriter(v, pluginID)
		},
		Cache:      cache,
		Written:    hooks.Written,
		Vaults:     registry.Vaults.List,
		Find:       findVault(registry),
		IndexState: hooks.IndexState,
		Moved:      hooks.Moved,
		Index:      index,
	}
	if c.VaultWatcher != nil {
		vaults.Watcher = vault.NewWatchChanges(c.VaultWatcher).Execute
	}
	if vaults.IndexState == nil {
		vaults.IndexState = getIndexState(db)
	}
	if vaults.Moved == nil {
		vaults.Moved = c.Moves
	}
	return vaults.Access(), nil
}

func findVault(registry vaultregistry.Ports) func(id domain.VaultID) (domain.Vault, error) {
	return func(id domain.VaultID) (domain.Vault, error) {
		found, isFound, err := registry.Lookup.FindVault(string(id))
		if err != nil {
			return domain.Vault{}, err
		}
		if !isFound {
			return domain.Vault{}, fmt.Errorf("vault %q: %w", id, fs.ErrNotExist)
		}
		return found, nil
	}
}

func getIndexState(db *vault.Index) func(ctx context.Context, v domain.Vault) (plugindomain.IndexState, error) {
	return func(ctx context.Context, v domain.Vault) (plugindomain.IndexState, error) {
		has, err := db.VaultHolds().Has(ctx, v.ID)
		if err != nil {
			return plugindomain.IndexState{}, err
		}
		if has {
			return plugindomain.IndexState{Status: plugindomain.Indexed}, nil
		}
		return plugindomain.IndexState{Status: plugindomain.Indexing}, nil
	}
}

func getPluginDir(c pluginsWiring, id string) (string, bool) {
	if c.PluginHost == nil {
		return "", false
	}
	p, ok := c.PluginHost.GetPlugin(id)
	if !ok {
		return "", false
	}
	return p.Dir, true
}

func getFrontendPluginIDs(c pluginsWiring) []string {
	if c.PluginHost == nil {
		return nil
	}
	infos := c.PluginHost.GetPluginInfos()
	ids := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.HasFrontend {
			continue
		}
		ids = append(ids, info.ID)
	}
	return ids
}

func setPluginWindow(c pluginsWiring, window hostendpoint.WindowControl) {
	if c.PluginHost != nil {
		c.PluginHost.SetPluginWindow(window)
	}
}

func emitPluginEvent(c pluginsWiring, event string) {
	if c.PluginHost != nil {
		var emitter pluginport.PluginEventEmitter = c.PluginHost
		emitter.EmitEvent(event)
	}
}

// credentialsLocator is a key store that keeps its values in a file when the machine has no keychain.
type credentialsLocator interface {
	// GetCredentialsPath returns the file the values are kept in, or the empty text while the
	// system keychain is in use.
	GetCredentialsPath(ctx context.Context) string
}

func openSecretLedger(c pluginsWiring) *settingsadapter.SecretLedger {
	return settingsadapter.NewSecretLedger(c.SettingsFiles, c.SettingsFile, c.ReadCheckedSettings, c.RefreshSettingEvents)
}

func openSecrets(c pluginsWiring) *secret.Secrets {
	var plugins pluginport.SecretPlugins = textbridge.NoSecretPlugins{}
	if c.PluginHost != nil {
		plugins = c.PluginHost
	}
	return secret.NewSecrets(openSecretLedger(c), c.KeyStore, plugins, newSecretID, c.CredentialsPath, c.SecretGate)
}

func getCredentialsPath(c pluginsWiring) string {
	locator, isLocator := c.KeyStore.(credentialsLocator)
	if !isLocator {
		return ""
	}
	return locator.GetCredentialsPath(context.Background())
}

func newSecretID() (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("read random bytes for the id of a secret: %w", err)
	}
	return "s" + hex.EncodeToString(random[:]), nil
}

func setSecretSource(c pluginsWiring) {
	if c.PluginHost == nil {
		return
	}
	ledger := openSecretLedger(c)
	c.PluginHost.SetSecretSource(plugindomain.SecretSource{
		List:  ledger.GetSecrets,
		Prune: openSecrets(c).Prune.Execute,
	})
}

func signalPluginChanges(c pluginsWiring) {
	if c.PluginChanges != nil {
		c.PluginChanges.Signal()
	}
}

func watchPluginChanges(c pluginsWiring) (<-chan struct{}, func()) {
	if c.PluginChanges == nil {
		return make(chan struct{}), func() {}
	}
	return c.PluginChanges.Subscribe()
}

func openChooseTabKind(c pluginsWiring) *tabkind.Choose {
	if c.PluginHost == nil {
		return nil
	}
	choose, err := tabkind.NewChoose(manifestlist.Open(c.PluginHost), c.PluginHost, getWhenConfig(c))
	if err != nil {
		pluginsHandleError(c, err)
		return nil
	}
	return choose
}

func openVersionItems(c pluginsWiring) *textversion.VersionItems {
	if c.TextVersions == nil {
		return nil
	}
	var commands pluginport.PluginCommands = manifestlist.NoManifests{}
	var runs pluginport.PluginCommandRuns = commandbridge.NoCommands{}
	var types contenttypeport.ContentTypeTable = sharedmimeinfo.Open()
	if c.PluginHost != nil && !c.SafeMode {
		commands, runs, types = manifestlist.Open(c.PluginHost), commandbridge.Open(c.PluginHost), c.PluginHost
	}
	run := plugincommand.NewCommands(commands, runs, types, pluginsNewNoteReader(c), getWhenConfig(c)).Run
	items, err := textversion.NewVersionItems(c.TextVersions, run, types)
	if err != nil {
		pluginsHandleError(c, err)
		return nil
	}
	return items
}

func openTriggers(c pluginsWiring) *trigger.Triggers {
	if c.PluginHost == nil || c.SafeMode || c.TextVersions == nil {
		return nil
	}
	return trigger.NewTriggers(trigger.TriggersDeps{
		Plugins: manifestlist.Open(c.PluginHost), Run: getPluginCommands(c).Run, Readers: c.VaultReaders,
		Notes: pluginsNewNoteReader(c), Types: c.PluginHost, Versions: c.TextVersions, Config: getWhenConfig(c),
	})
}

func pluginsNewNoteReader(c pluginsWiring) plugincommand.NoteReader {
	if c.VaultReaders == nil {
		return nil
	}
	return note.NewReadFrontmatter(c.VaultReaders)
}

func getWhenConfig(c pluginsWiring) pluginport.WhenConfig {
	settings := settingsadapter.WhenConfig{App: c.ListAppSettings}
	if c.PluginHost != nil {
		settings.Plugins = c.PluginHost.GetPluginInfos
	}
	return settings
}

func getProblemChecks(u *vault.GetProblems) func(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	return func(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
		report, err := u.Execute(ctx, v, named...)
		return report.Ran, report.Found, err
	}
}
