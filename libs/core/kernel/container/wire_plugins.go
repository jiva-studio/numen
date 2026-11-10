package container

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/completion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// SetSafeMode is the configuration started in safe mode or out of it, and its plugin host with it.
func (c Config) SetSafeMode(enabled bool) Config {
	c.SafeMode = setSafeMode(plugins(c), enabled).SafeMode
	return c
}

// SetPluginVaultPath is the vault the plugins' paths are inside.
func SetPluginVaultPath(c Config, path string) {
	setPluginVaultPath(plugins(c), path)
}

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

func openedPluginWork(c Config, db *vault.Index, v domain.Vault) (Config, error) {
	opened, _, err := openPluginWork(c, db, v)
	return opened, err
}

func closeAfterWork(ctx context.Context, work *run.PluginWork, closeIndex func() error) func() error {
	return func() error {
		return errors.Join(work.WaitIdle(ctx), closeIndex())
	}
}

func openPluginWork(c Config, db *vault.Index, v domain.Vault) (Config, *run.PluginWork, error) {
	setPluginVaultPath(plugins(c), v.Path)
	versions, err := newTextVersions(vaultOf(c), db)
	if err != nil {
		return c, nil, err
	}
	c.TextVersions = versions
	registry, err := c.Registry()
	if err != nil {
		return c, nil, err
	}
	current := func() (domain.Vault, bool) { return v, true }
	if err := installPluginVault(plugins(c), registry, db, current, PluginVaultHooks{}); err != nil {
		return c, nil, err
	}
	work := pluginsOpenPluginWork(plugins(c), run.NewTasks(), &run.Lock{})
	setPluginTexts(vaultOf(c))
	return c, work, nil
}

// GetPluginServices is the forwarding of the services plugins serve to the windows.
func GetPluginServices(c Config) PluginServices {
	return getPluginServices(plugins(c))
}

// InstallPluginVault lets the plugins work through the host the vault of a list and the index that
// holds them: the one current names, else the one a call names by identity.
func InstallPluginVault(c Config,
	vaults vaultregistry.Ports, db *vault.Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) error {
	return installPluginVault(plugins(c), vaults, db, current, hooks)
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
