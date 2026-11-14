package container

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
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
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/servicebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

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
}

// pluginsWiring wires the plugin catalog, plugin host and dependencies over the settings and
// the vaults the plugins work with.
type pluginsWiring struct {
	pluginInstall
	settings settingsWiring
	search   searchWiring
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
// plugins failed, with the failure. report receives the problems the host meets outside a call, such as a failed
// shutdown of a plugin.
func NewPluginHost(report func(error), target string, dirs ...string) (*PluginHost, error) {
	host, err := OpenPluginRoles(report, target, dirs...)
	return adaptPluginHost(host), err
}

// OpenPluginRoles creates the roles of a plugin host as NewPluginHost does and returns them, for a caller that holds
// the roles. The roles are returned also when the discovery of the plugins failed, with the failure. report receives
// the problems the host meets outside a call.
func OpenPluginRoles(report func(error), target string, dirs ...string) (*PluginRoles, error) {
	host := NewPluginRoles(clock.System{}, target, dirs...)
	host.Failures.SetErrorHandler(report)
	host.ContentTypes.SetContentTypeTable(contenttype.Open())
	host.Dependencies.Replace(newDependencyRegistry(""))
	return host, host.Discovery.Discover()
}

func setPluginVaultPath(c pluginsWiring, path string) {
	c.PluginHost.Runtime.SetVaultPath(path)
}

func stopPlugins(c pluginsWiring, ctx context.Context) error {
	return c.PluginHost.Watch.TeardownAll(ctx, pluginGracePeriod)
}

func getPluginCommands(c pluginsWiring) PluginCommandUseCases {
	return newPluginCommands(c, c.PluginHost, nil)
}

func newPluginCommands(c pluginsWiring, host *PluginHost, versions plugincommand.VersionItems) PluginCommandUseCases {
	// versions are the text versions that a command with items lists and runs over, nil where the window has none.
	commands, conditions := pluginmanifest.New(host.Runtime), when.NewConditions()
	list := plugincommand.NewList(commands, conditions)
	run := plugincommand.NewRun(commands, commandbridge.Open(host.Runtime), host.Watch, pluginsNewNoteReader(c), getWhenConfig(c), conditions)
	return PluginCommandUseCases{
		List:       list,
		Listing:    plugincommand.NewListCommands(list, host.Watch),
		Run:        run,
		RunCommand: plugincommand.NewRunCommand(list, run, versions),
		ListItems:  plugincommand.NewListItems(list, versions),
	}
}

func getPluginServices(c pluginsWiring) editor.PluginServices {
	return servicebridge.Open(c.PluginHost.Runtime, c.PluginHost.Runtime)
}

func getPluginRetrier(c pluginsWiring) func(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
	return c.PluginHost.Runtime.RetryPlugin
}

func signalFailureClears(c pluginsWiring) {
	c.PluginHost.Runtime.SetOnFailureCleared(func(string) { signalPluginChanges(c) })
}

func setPluginWork(c pluginsWiring, tasks *run.Tasks, models *run.Lock) {
	pluginsOpenPluginWork(c, tasks, models)
}

func pluginsOpenPluginWork(c pluginsWiring, tasks *run.Tasks, models *run.Lock) *run.PluginWork {
	work := run.NewPluginWork(models, tasks, func(pluginID string) string { return pluginsGetPluginName(c, pluginID) })
	c.PluginHost.Lender.Update(func(l *hostendpoint.Lent) { l.Work = work })
	return work
}

func pluginsGetPluginName(c pluginsWiring, pluginID string) string {
	if found, isKnown := c.PluginHost.Runtime.GetPlugin(pluginID); isKnown && found.Manifest.Name != "" {
		return found.Manifest.Name
	}
	return pluginID
}

func setPluginVault(c pluginsWiring, v *pluginvault.Access) {
	c.PluginHost.Lender.Update(func(l *hostendpoint.Lent) { l.Vault = v })
}

func setPluginVaultList(c pluginsWiring, list pluginvault.List) {
	c.PluginHost.Lender.Update(func(l *hostendpoint.Lent) { l.VaultList = list })
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
	runtimeConfig.Offline = provision.IsOffline()
	return provision.NewRegistry(
		onnxruntime.NewONNXProvisioner(runtimeConfig),
		provision.NewFileProvisioner(provision.Config{Dir: runtimeConfig.Dir, Offline: runtimeConfig.Offline, Clock: clock.System{}}),
	)
}
