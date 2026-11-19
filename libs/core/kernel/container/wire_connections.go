package container

import (
	"cmp"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	windowconnection "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connectioncall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/rowoptions"
)

func hasPluginHost(settingsDeps settingsWiring) bool {
	return settingsDeps.Views.Types != nil
}

func openConnectionRepository(c settingsWiring) *settingsadapter.ConnectionRepository {
	return settingsadapter.NewConnectionRepository(
		c.SettingsFiles,
		func() (string, error) { return settingsSettingsFile(c) },
		func() (settingsadapter.File, error) { return readCheckedSettings(c) },
		func() { refreshSettingEvents(c) },
	)
}

func openConnectionRows(install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) *settingsadapter.ConnectionRows {
	s := settingsDeps
	return settingsadapter.NewConnectionRows(settingsadapter.ConnectionRowsSource{
		Connections: openConnectionRepository(s),
		Detected:    openDetection(install, settingsDeps, vaultDeps),
		Plugins:     settingsPlugins(s),
		Declared:    getDeclarations(s),
		ReadStored:  settingsStore(s).ReadUncheckedFile,
		Write:       getSettingsWriter(s),
		Refresh:     settingsDeps.Configuration.Refresh,
		Report:      s.ErrorHandler,
	})
}

func openConnectionDeps(install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) (connection.Deps, bool) {
	if !hasPluginHost(settingsDeps) {
		return connection.Deps{}, false
	}
	deps := connection.NewDeps(openConnectionRepository(settingsDeps), settingsDeps.Views.Types, install.KeyStore, openConnectionRows(install, settingsDeps, vaultDeps), openDetection(install, settingsDeps, vaultDeps))
	if options := openRowOptions(install, settingsDeps, deps); options != nil {
		deps = deps.AddOptionLists(options)
	}
	return deps, true
}

func openDetection(install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) connectionport.Detected {
	detection := install.Endpoints.Detection
	if detection == nil {
		source := providerbridge.NewQuestionSource(install.Execution.Runtime.Activator)
		return providerbridge.NewDetection(settingsDeps.Views.Types, source, vaultDeps.Clock)
	}
	detection.Follow(func() { refreshSettingEvents(settingsDeps) }, settingsDeps.ErrorHandler)
	return detection
}

func openConnectionUseCases(install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) *windowconnection.UseCases {
	deps, ok := openConnectionDeps(install, settingsDeps, vaultDeps)
	if !ok {
		return nil
	}
	return &windowconnection.UseCases{
		Edits: windowconnection.Edits{
			Create: connection.NewCreate(deps, settingsadapter.CreateConnectionID), Update: connection.NewUpdate(deps),
			Delete: connection.NewDelete(deps),
		},
		Reads: windowconnection.Reads{
			List: connection.NewList(deps), ListTypes: connection.NewListTypes(deps, cmp.Or[connectionport.OperationTitles](settingsDeps.Views.Operations, none.OperationTitles{})),
			RowOptions: openRowOptions(install, settingsDeps, deps),
		},
	}
}

func openModelCalls(install pluginInstall, settingsDeps settingsWiring) connectionport.ModelCalls {
	return getForwardedCalls(settingsDeps.Views.Types, install.Endpoints.Endpoint)
}

func setModelCalls(install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring, notices connectionport.ReportSetAside) {
	deps, ok := openConnectionDeps(install, settingsDeps, vaultDeps)
	if !ok {
		return
	}
	calls := modelcall.NewCalls(
		modelcall.Connections{
			Resolve: connection.NewResolve(deps).Execute, ReadSecrets: connection.NewReadSecrets(deps).Execute,
			Rows:   connection.NewListRows(deps, openConnectionRows(install, settingsDeps, vaultDeps)).Execute,
			Titles: settingsDeps.Views.Operations,
		},
		openCallSource(install, settingsDeps, deps), vaultDeps.Clock, vaultDeps.Clock.Wait, notices,
	)
	lists := openRowLists(install, settingsDeps, vaultDeps)
	install.Endpoints.Lender.Update(func(l *hostendpoint.Lent) {
		l.ModelCalls = calls
		if lists != nil {
			l.RowLists = lists
		}
	})
}

// GetConnectionCalls is the calls plugin views make through their connections, for a window that serves
// ConnectionCallService without the routes of the editor. It is nil where the installation runs no
// plugins.
func GetConnectionCalls(prefs *PluginPreferences, endpoints *PluginEndpoints) connectionport.ModelCalls {
	if prefs == nil || endpoints == nil {
		return nil
	}
	return getForwardedCalls(prefs.Views.Types, endpoints.Endpoint)
}

func getForwardedCalls(types connectionport.Types, endpoint *hostendpoint.Services) connectionport.ModelCalls {
	if types == nil {
		return nil
	}
	return modelcall.NewForwardedCalls(func() connectionport.ModelCalls { return endpoint.GetModelCalls() })
}

func openRowLists(install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) connectioncall.Rows {
	deps, ok := openConnectionDeps(install, settingsDeps, vaultDeps)
	if !ok {
		return nil
	}
	resolve := connection.NewResolve(deps).Execute
	options := openRowOptions(install, settingsDeps, deps)
	if options == nil {
		return nil
	}
	return modelcall.NewRows(resolve, connection.NewListRows(deps, openConnectionRows(install, settingsDeps, vaultDeps)).Execute, options.ListOptions)
}

// GetConnectionRows is the listing of the rows plugin views offer a choice of, for a window that serves
// ConnectionCallService without the routes of the editor. It is nil where the installation runs no plugins.
func GetConnectionRows(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources) connectioncall.Rows {
	install, settingsDeps, _, vaultDeps := plugins(c, exec, prefs, endpoints, res)
	return openRowLists(install, settingsDeps, vaultDeps)
}

func openRowOptions(install pluginInstall, settingsDeps settingsWiring, deps connection.Deps) *rowoptions.Options {
	if install.Endpoints.RowOptions == nil || install.Endpoints.RunningProviders == nil {
		return nil
	}
	return rowoptions.NewOptions(rowoptions.Deps{
		Resolve: connection.NewResolve(deps).Execute, ReadSecrets: connection.NewReadSecrets(deps).Execute,
		Source: providerbridge.NewQuestionSource(install.Execution.Runtime.Activator), Running: install.Endpoints.RunningProviders,
		Lists: install.Endpoints.RowOptions, Titles: settingsDeps.Views.Operations,
		Grants: connection.NewGetGrantRevision(deps).Execute,
	})
}

func openCallSource(install pluginInstall, settingsDeps settingsWiring, deps connection.Deps) connectionport.ProviderSource {
	var source connectionport.ProviderSource = providerbridge.NewSource(install.Execution.Runtime.Activator)
	if install.Endpoints.Detection != nil {
		source = providerbridge.NewRedetectingSource(source, install.Endpoints.Detection)
	}
	if options := openRowOptions(install, settingsDeps, deps); options != nil {
		source = providerbridge.NewCallReportingSource(source, options.RefreshRunning)
	}
	return source
}
