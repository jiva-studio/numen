package container

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/license"

	pluginvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"

	windowlogs "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"

	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

func reportDependencies(watch dependencyReporter, tasks *run.Tasks, record func(domainlogs.Entry)) {
	show := run.NewDependencyTasks(tasks)
	watch.ObserveDependencies(func(req manifest.DependencyRequirement, status plugindomain.DependencyStatus) {
		show.Execute(req, status, func() { watch.CancelDependency(req) })
		supervision.RecordDependencyStatus(record, req.ID, status)
	})
}

// pluginInstall supplies storage, catalog and host endpoints to the installation.
type pluginInstall struct {
	Licenses        license.Declarations
	Configuration   configRefresher
	OperationTitles connection.OperationTitles
	PluginCachePath string
	ServiceDir      string
	SafeMode        bool
	Endpoints       *PluginEndpoints
	PluginCatalog   catalog.Config
	KeyStore        pluginport.KeyStore
}

func getPluginCatalog(cfg catalog.Config, report func(error)) *catalog.Service {
	if cfg.Target == "" {
		return nil
	}
	service, err := catalog.NewService(cfg)
	if err != nil {
		report(err)
		return nil
	}
	return service
}

// pluginGracePeriod is how long a plugin has to exit after a stop request.
const pluginGracePeriod = 2 * time.Second

// errSafeMode is an installation started in safe mode, which starts no plugin.
var errSafeMode = errors.New("numen is running in safe mode")

// NewPluginHost creates and initializes a plugin host for the target and search directories on the installation clock.
// The host provisions the dependencies the plugins declare. The host is returned also when the discovery of the
// plugins failed, with the failure. report receives the problems the host meets outside a call, such as a failed
// shutdown of a plugin.
func NewPluginHost(indexPath string, res *Resources, report func(error), target string, dirs ...string) (*PluginExecution, *PluginPreferences, *PluginEndpoints, error) {
	management, supervision := NewPluginRoles(GetClock(res), target, dirs...)
	err := InitializePluginRoles(indexPath, res, management, supervision, report)
	exec, prefs, endpoints := CreatePluginHost(management, supervision)
	return exec, prefs, endpoints, err
}

// InitializePluginRoles installs host services before discovering the target's plugins.
func InitializePluginRoles(indexPath string, res *Resources, management *PluginManagement, processes *PluginSupervision, report func(error)) error {
	record := openOptionalRecordLog(indexPath, res)
	processes.Failures.SetOnFailure(supervision.ReportPluginFailure(record))
	processes.Failures.SetErrorHandler(windowlogs.ReportError(record, report, "Plugin host failure"))
	management.ContentTypes.SetContentTypeTable(contenttype.Open())
	processes.Dependencies.Replace(newDependencyRegistry(processes.Clock, processes.Services.Observe, ""))
	err := management.Discovery.Discover()
	windowlogs.RecordError(record, "Plugin discovery failed", err)
	return err
}

// StopPlugins cancels pending dependencies before stopping plugin processes.
func StopPlugins(watch PluginWatch, ctx context.Context) error {
	watch.CancelDependencies()
	return watch.TeardownAll(ctx, pluginGracePeriod)
}

func newPluginCommands(config pluginport.WhenConfig, vaultDeps vaultWiring, host *PluginExecution, versions plugincommand.VersionItems) pluginCommandUseCases {
	// versions are the text versions that a command with items lists and runs over, nil where the window has none.
	commands, conditions := host.Runtime.Manifests, when.NewConditions()
	list := plugincommand.NewList(commands, commands, conditions)
	run := plugincommand.NewRunner(commands, commandbridge.Open(host.Runtime.Activator), host.Watch.ContentTypeTable, vaultDeps.Readers, config, conditions)
	return pluginCommandUseCases{
		List:       list,
		Listing:    plugincommand.NewListCommands(list, host.Watch.ContentTypeTable),
		Run:        run,
		RunCommand: plugincommand.NewRunCommand(list, run, versions),
		ListItems:  plugincommand.NewListItems(list, versions),
	}
}

func openPluginWork(install pluginInstall, exec *PluginExecution, tasks *run.Tasks) *run.PluginWork {
	work := run.NewPluginWork(&install.Endpoints.workLock, tasks, func(pluginID string) string {
		return getPluginName(exec.Runtime.Resolver, pluginID)
	})
	install.Endpoints.Lender.Update(func(l *hostendpoint.Lent) { l.Work = work })
	return work
}

func getPluginName(resolver pluginResolver, pluginID string) string {
	if found, isKnown := resolver.GetPlugin(pluginID); isKnown && found.Manifest.Name != "" {
		return found.Manifest.Name
	}
	return pluginID
}

func setPluginVault(lender *hostendpoint.Lender, v *pluginvault.Access) {
	lender.Update(func(l *hostendpoint.Lent) { l.Vault = v })
}

func setPluginVaultList(lender *hostendpoint.Lender, list pluginvault.List) {
	lender.Update(func(l *hostendpoint.Lent) { l.VaultList = list })
}

func newDependencyRegistry(clock machineport.Clock, observe func(manifest.DependencyRequirement, plugindomain.DependencyStatus), defaultDir string) supervision.Dependencies {
	if provision.IsMockML() {
		return provision.NewObservedRegistry(observe, provision.NewPlaceholderProvisioner(provision.GetPlaceholderDir()))
	}
	return provision.NewObservedRegistry(observe, provision.NewFileProvisioner(provision.Config{
		Dir: provision.GetModelsDir(defaultDir), Offline: provision.IsOffline(), Clock: clock,
	}))
}

func getDependencyProvisioner(services *supervision.Services) func() pluginport.DependencyRegistry {
	return func() pluginport.DependencyRegistry {
		if dependencies := services.GetDependencies(); dependencies != nil {
			return dependencies
		}
		return nil
	}
}
