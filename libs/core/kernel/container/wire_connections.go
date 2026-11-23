package container

import (
	"cmp"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	windowconnection "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

func openConnectionRepository(c settingsWiring) *settingsadapter.ConnectionRepository {
	return settingsadapter.NewConnectionRepository(
		c.SettingsFiles,
		func() (string, error) { return getSettingsFile(c) },
		func() (settingsadapter.File, error) { return createSettingsStore(c).ReadCheckedFile() },
		func() { getFeed(c).Refresh() },
	)
}

func openConnectionRows(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) *settingsadapter.ConnectionRows {
	s := settingsDeps
	return settingsadapter.NewConnectionRows(settingsadapter.ConnectionRowsSource{
		Connections: openConnectionRepository(s),
		Detected:    openDetection(install, exec, settingsDeps, vaultDeps),
		ListPlugins: s.Infos.GetPluginInfos,
		Declared:    getDeclarations(s),
		ReadStored:  createSettingsStore(s).ReadUncheckedFile,
		Write:       getSettingsWriter(s),
		Refresh:     install.Configuration.Refresh,
		Report:      s.HandleError,
	})
}

func openConnectionDeps(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) (connection.Deps, bool) {
	if settingsDeps.ConnectionTypes == nil {
		return connection.Deps{}, false
	}
	deps := connection.NewDeps(openConnectionRepository(settingsDeps), settingsDeps.ConnectionTypes, install.KeyStore,
		openConnectionRows(install, exec, settingsDeps, vaultDeps), openDetection(install, exec, settingsDeps, vaultDeps))
	if options := openRowOptions(install, exec, deps); options != nil {
		deps = deps.AddOptionLists(options)
	}
	return deps, true
}

func openDetection(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) connectionport.Detected {
	detection := install.Endpoints.Detection
	if detection == nil {
		source := providerbridge.NewQuestionSource(exec.Runtime.Activator)
		return providerbridge.NewDetection(settingsDeps.ConnectionTypes, source, vaultDeps.Clock)
	}
	detection.Follow(func() { getFeed(settingsDeps).Refresh() }, settingsDeps.HandleError)
	return detection
}

func openConnectionUseCases(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) *windowconnection.UseCases {
	deps, ok := openConnectionDeps(install, exec, settingsDeps, vaultDeps)
	if !ok {
		return nil
	}
	return &windowconnection.UseCases{
		Edits: windowconnection.Edits{
			Create: connection.NewCreate(deps, settingsadapter.CreateConnectionID), Update: connection.NewUpdate(deps),
			Delete: connection.NewDelete(deps),
		},
		Reads: windowconnection.Reads{
			List: connection.NewList(deps), ListTypes: connection.NewListTypes(deps, cmp.Or[connection.OperationTitles](install.OperationTitles, none.OperationTitles{})),
			RowOptions: windowconnection.NewRowOptions(openRowOptions(install, exec, deps)),
		},
	}
}

func setModelCalls(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring, notices connection.ReportSetAside) {
	deps, ok := openConnectionDeps(install, exec, settingsDeps, vaultDeps)
	if !ok {
		return
	}
	calls := connection.NewCalls(
		connection.Connections{
			Resolve: connection.NewResolve(deps).Execute, ReadSecrets: connection.NewReadSecrets(deps).Execute,
			Rows:   connection.NewListRows(deps, openConnectionRows(install, exec, settingsDeps, vaultDeps)).Execute,
			Titles: install.OperationTitles,
		},
		openCallSource(install, exec, deps), vaultDeps.Clock, vaultDeps.Clock.Wait, notices,
	)
	lists := openRowLists(install, exec, settingsDeps, vaultDeps)
	install.Endpoints.Lender.Update(func(l *hostendpoint.Lent) {
		l.ModelCalls = calls
		if lists != nil {
			l.RowLists = lists
		}
	})
}

func getForwardedCalls(types connectionport.Types, endpoint *hostendpoint.Services) connectionport.ModelCalls {
	if types == nil {
		return nil
	}
	return connection.NewForwardedCalls(func() connectionport.ModelCalls { return endpoint.GetModelCalls() })
}

func openRowLists(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) *connection.Rows {
	deps, ok := openConnectionDeps(install, exec, settingsDeps, vaultDeps)
	if !ok {
		return nil
	}
	resolve := connection.NewResolve(deps).Execute
	options := openRowOptions(install, exec, deps)
	if options == nil {
		return nil
	}
	return connection.NewRows(resolve, connection.NewListRows(deps, openConnectionRows(install, exec, settingsDeps, vaultDeps)).Execute, options.ListOptions)
}

func openRowOptions(install pluginInstall, exec *PluginExecution, deps connection.Deps) *connection.Options {
	if install.Endpoints.RowOptions == nil || install.Endpoints.RunningProviders == nil {
		return nil
	}
	return connection.NewOptions(connection.OptionsDeps{
		Resolve: connection.NewResolve(deps).Execute, ReadSecrets: connection.NewReadSecrets(deps).Execute,
		Source: providerbridge.NewQuestionSource(exec.Runtime.Activator), Running: install.Endpoints.RunningProviders,
		Lists: install.Endpoints.RowOptions, Titles: install.OperationTitles,
		Grants: connection.NewGetGrantRevision(deps).Execute,
	})
}

func openCallSource(install pluginInstall, exec *PluginExecution, deps connection.Deps) connectionport.ProviderSource {
	var source connectionport.ProviderSource = providerbridge.NewSource(exec.Runtime.Activator)
	if install.Endpoints.Detection != nil {
		source = providerbridge.NewRedetectingSource(source, install.Endpoints.Detection)
	}
	if options := openRowOptions(install, exec, deps); options != nil {
		source = providerbridge.NewCallReportingSource(source, options.RefreshRunning)
	}
	return source
}
