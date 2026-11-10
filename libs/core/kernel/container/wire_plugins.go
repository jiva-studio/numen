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

func (c Config) reportDependencies(tasks *run.Tasks) {
	reportDependencies(c.plugins(), tasks)
}

func (c Config) getPluginCatalog() *catalog.Service {
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

func (c Config) getPluginCommands() plugincommand.Commands {
	return getPluginCommands(c.plugins())
}

func (c Config) openPluginCommands(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return plugincommand.Run{}, nil, err
	}
	_, work, err := c.openPluginWork(db, v)
	if err != nil {
		return plugincommand.Run{}, nil, errors.Join(err, db.Close())
	}
	return c.getPluginCommands().Run, closeAfterWork(ctx, work, db.Close), nil
}

func (c Config) openedPluginWork(ctx context.Context, db *vault.Index, v domain.Vault) (Config, error) {
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

func (c Config) getPluginRetrier() func(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
	return getPluginRetrier(c.plugins())
}

func (c Config) reportPluginFailures(tasks *run.Tasks) {
	reportPluginFailures(c.plugins(), tasks)
}

// SetPluginWork opens the lock the models are held under and the list of tasks to the plugins.
func (c Config) SetPluginWork(tasks *run.Tasks, models *run.Lock) {
	setPluginWork(c.plugins(), tasks, models)
}

func (c Config) getPluginName(pluginID string) string {
	return pluginsGetPluginName(c.plugins(), pluginID)
}

func (c Config) setPluginVaultList(list hostendpoint.VaultList) {
	setPluginVaultList(c.plugins(), list)
}

func (c Config) openPluginCache() (pluginport.PluginCache, error) {
	return openPluginCache(c.plugins())
}

func (c Config) GetPluginDir(id string) (string, bool) {
	return getPluginDir(c.plugins(), id)
}

func (c Config) GetFrontendPluginIDs() []string {
	return getFrontendPluginIDs(c.plugins())
}

func (c Config) setPluginWindow(window hostendpoint.WindowControl) {
	setPluginWindow(c.plugins(), window)
}

func (c Config) openSecrets() *secret.Secrets {
	return openSecrets(c.plugins())
}

func (c Config) providerOfAgents() agentport.AgentProvider {
	return pluginsAgentProvider(c.plugins())
}

func (c Config) getSettingsAgent() agentport.Agent {
	return getSettingsAgent(c.plugins())
}

func (c Config) openChooseTabKind() *tabkind.Choose {
	return openChooseTabKind(c.plugins())
}

func (c Config) watchPluginChanges() (<-chan struct{}, func()) {
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
	ApplySetting        settingsport.PluginSettingApplier
	SetCompleter        func(completer hostendpoint.Completer)
	SetPluginWork       func(work hostendpoint.WorkQueue)
	SetPluginVault      func(vault *hostendpoint.VaultAccess)
	GetPluginVault      func() *hostendpoint.VaultAccess
	SetPluginTexts      func(texts hostendpoint.TextVersions)
	SetPluginSearch     func(search *hostendpoint.SearchAccess)
	SetPluginWindow     func(window hostendpoint.WindowControl)
	SetPluginVaultList  func(list hostendpoint.VaultList)
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
		SetCompleter:        host.Services.SetCompleter,
		SetPluginWork:       host.Services.SetPluginWork,
		SetPluginVault:      host.Services.SetPluginVault,
		GetPluginVault:      host.Services.GetPluginVault,
		SetPluginTexts:      host.Services.SetPluginTexts,
		SetPluginSearch:     host.Services.SetPluginSearch,
		SetPluginWindow:     host.Services.SetPluginWindow,
		SetPluginVaultList:  host.Services.SetPluginVaultList,
		SetStorage:          func(cacheRoot, dataRoot string) { setStorage(host, cacheRoot, dataRoot) },
		RefreshDependencies: func() { host.Services.SetDependencies(GetDependencyRegistry("")) },
	}
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

