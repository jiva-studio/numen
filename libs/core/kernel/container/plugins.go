package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/assembly"
	pluginswiring "github.com/jiva-studio/numen/libs/core/kernel/container/plugins"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
)

const envMockML = pluginswiring.EnvMockML

const programsLimit = pluginswiring.ProgramsLimit

// PluginServices is where a window forwards the calls it receives for the services plugins serve.
type PluginServices = pluginswiring.PluginServices

// PluginVaultHooks is what a window adds to the vault it lets the plugins work.
type PluginVaultHooks = pluginswiring.PluginVaultHooks

// NewPluginHost creates and initializes a plugin host for the specified target and search directories.
func NewPluginHost(target string, dirs ...string) *plugin.Host {
	return pluginswiring.NewPluginHost(target, dirs...)
}

// NewDependencies returns the provisioners as the environment names them now.
func NewDependencies() pluginport.DependencyCatalog {
	return pluginswiring.NewDependencies()
}

// NewDependenciesIn is NewDependencies with the dependencies kept in dir where NUMEN_MODELS_DIR names no folder.
func NewDependenciesIn(dir string) pluginport.DependencyCatalog {
	return pluginswiring.NewDependenciesIn(dir)
}

func newDependencies(defaultDir string) *provision.Registry {
	reg, ok := pluginswiring.NewDependenciesIn(defaultDir).(*provision.Registry)
	if !ok {
		return nil
	}
	return reg
}

// ReportDependencies keeps one task in the list for each dependency being provisioned.
func (c Config) ReportDependencies(tasks *run.Tasks) {
	c.plugins().ReportDependencies(tasks)
}

// GetPluginCatalog is the catalog of the plugins on disk, the disabled ones too, and of the sources
// the configuration names.
func (c Config) GetPluginCatalog() *catalog.Service {
	return c.plugins().GetPluginCatalog()
}

func (c Config) getEditorCatalog() assembly.PluginCatalog {
	return c.plugins().GetEditorCatalog()
}

func (c Config) resolvePlugin(capability string) (pluginport.PluginManager, error) {
	return c.plugins().ResolvePlugin(capability)
}

// SetSafeMode is the configuration started in safe mode or out of it, and its plugin host with it.
func (c Config) SetSafeMode(enabled bool) Config {
	c.SafeMode = c.plugins().SetSafeMode(enabled).SafeMode
	return c
}

// SetPluginVaultPath is the vault the plugins' paths are inside.
func (c Config) SetPluginVaultPath(path string) {
	c.plugins().SetPluginVaultPath(path)
}

// StopPlugins stops every plugin the host runs.
func (c Config) StopPlugins(ctx context.Context) error {
	return c.plugins().StopPlugins(ctx)
}

// GetPluginCommands is the use cases over the commands the plugins declare.
func (c Config) GetPluginCommands() plugincommand.Commands {
	return c.plugins().GetPluginCommands()
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
func (c Config) OpenPluginWork(ctx context.Context, db *Index, v domain.Vault) (Config, error) {
	opened, _, err := c.openPluginWork(db, v)
	return opened, err
}

func closeAfterWork(ctx context.Context, work *run.PluginWork, closeIndex func() error) func() error {
	return func() error {
		return errors.Join(work.WaitIdle(ctx), closeIndex())
	}
}

func (c Config) openPluginWork(db *Index, v domain.Vault) (Config, *run.PluginWork, error) {
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
	pluginVault, err := c.OpenPluginVault(registry, db, current, PluginVaultHooks{})
	if err != nil {
		return c, nil, err
	}
	work := c.plugins().OpenPluginWork(run.NewTasks(), &run.Lock{})
	c.SetPluginVault(pluginVault)
	c.SetPluginTexts()
	return c, work, nil
}

// GetPluginServices is the forwarding of the services plugins serve to the windows.
func (c Config) GetPluginServices() PluginServices {
	return c.plugins().GetPluginServices()
}

// GetPluginRetrier restarts, on request from the window, a plugin the host has suspended.
func (c Config) GetPluginRetrier() func(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
	return c.plugins().GetPluginRetrier()
}

// ReportPluginFailures keeps one task in the list for each plugin the host suspends.
func (c Config) ReportPluginFailures(tasks *run.Tasks) {
	c.plugins().ReportPluginFailures(tasks)
}

// SetPluginWork opens the lock the models are held under and the list of tasks to the plugins.
func (c Config) SetPluginWork(tasks *run.Tasks, models *run.Lock) {
	c.plugins().SetPluginWork(tasks, models)
}

func (c Config) getPluginName(pluginID string) string {
	return c.plugins().GetPluginName(pluginID)
}

// SetPluginVault is the vault the plugins use through the host.
func (c Config) SetPluginVault(v pluginport.PluginVault) {
	c.plugins().SetPluginVault(v)
}

// SetPluginVaultList sets the list of vaults exposed to plugins.
func (c Config) SetPluginVaultList(list pluginport.PluginVaultList) {
	c.plugins().SetPluginVaultList(list)
}

// OpenPluginCache is where a plugin keeps what it worked out for a vault.
func (c Config) OpenPluginCache() (pluginport.PluginCache, error) {
	return c.plugins().OpenPluginCache()
}

// OpenPluginVault is the vault the plugins work through the host over the vaults of a list and the
// index that holds them.
func (c Config) OpenPluginVault(
	registry vaultport.VaultRegistry, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) (pluginport.PluginVault, error) {
	return c.plugins().OpenPluginVault(registry, db, current, hooks)
}

// GetPluginDir is the folder of an installed plugin, and false for one not installed.
func (c Config) GetPluginDir(id string) (string, bool) {
	return c.plugins().GetPluginDir(id)
}

// GetFrontendPluginIDs are the plugins of the host's target that declare a frontend, in the order of
// their ids.
func (c Config) GetFrontendPluginIDs() []string {
	return c.plugins().GetFrontendPluginIDs()
}

// SetPluginWindow sets the window exposed to plugins.
func (c Config) SetPluginWindow(window pluginport.PluginWindow) {
	c.plugins().SetPluginWindow(window)
}

// OpenSecrets returns the use cases on the secrets shared between plugins.
func (c Config) OpenSecrets() *secret.Secrets {
	return c.plugins().OpenSecrets()
}

// AgentProvider runs tasks on the programs the plugins with the agent provider capability run.
func (c Config) AgentProvider() agentport.AgentProvider {
	return c.plugins().AgentProvider()
}

// GetSettingsAgent runs a window's tasks on the programs the plugins run, under the settings read at
// every task.
func (c Config) GetSettingsAgent() agentport.Agent {
	return c.plugins().GetSettingsAgent()
}

// OpenChooseTabKind is the choice of the kind of tab a note opens in, over the tab kinds of the
// enabled plugins.
func (c Config) OpenChooseTabKind() *tabkind.Choose {
	return c.plugins().OpenChooseTabKind()
}

// WatchPluginChanges reports each time the plugin host applied a batch of settings, which enables,
// disables and configures plugins, and each time it forgot the failure of a plugin, as after a retry
// that succeeded. The returned function stops the reports.
func (c Config) WatchPluginChanges() (<-chan struct{}, func()) {
	return c.plugins().WatchPluginChanges()
}
