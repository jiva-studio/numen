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

func hasPluginHost(c pluginsWiring) bool {
	return c.settings.Views.Types != nil
}

func openConnectionRepository(c settingsWiring) *settingsadapter.ConnectionRepository {
	return settingsadapter.NewConnectionRepository(
		c.SettingsFiles,
		func() (string, error) { return settingsSettingsFile(c) },
		func() (settingsadapter.File, error) { return readCheckedSettings(c) },
		func() { refreshSettingEvents(c) },
	)
}

func openConnectionRows(c pluginsWiring) *settingsadapter.ConnectionRows {
	s := c.settings
	return settingsadapter.NewConnectionRows(settingsadapter.ConnectionRowsSource{
		Connections: openConnectionRepository(s),
		Detected:    openDetection(c),
		Plugins:     settingsPlugins(s),
		Declared:    getDeclarations(s),
		ReadStored:  settingsStore(s).ReadUncheckedFile,
		Write:       getSettingsWriter(s),
		Refresh:     c.settings.Configuration.Refresh,
		Report:      s.ErrorHandler,
	})
}

func openConnectionDeps(c pluginsWiring) (connection.Deps, bool) {
	if !hasPluginHost(c) {
		return connection.Deps{}, false
	}
	deps := connection.NewDeps(openConnectionRepository(c.settings), c.settings.Views.Types, c.KeyStore, openConnectionRows(c), openDetection(c))
	if options := openRowOptions(c, deps); options != nil {
		deps = deps.AddOptionLists(options)
	}
	return deps, true
}

func openDetection(c pluginsWiring) connectionport.Detected {
	detection := c.Endpoints.Detection
	if detection == nil {
		source := providerbridge.NewQuestionSource(c.Execution.Runtime.Activator)
		return providerbridge.NewDetection(c.settings.Views.Types, source, c.vault.Clock)
	}
	detection.Follow(func() { refreshSettingEvents(c.settings) }, c.settings.ErrorHandler)
	return detection
}

func openConnectionUseCases(c pluginsWiring) *windowconnection.UseCases {
	deps, ok := openConnectionDeps(c)
	if !ok {
		return nil
	}
	return &windowconnection.UseCases{
		Edits: windowconnection.Edits{
			Create: connection.NewCreate(deps, settingsadapter.CreateConnectionID), Update: connection.NewUpdate(deps),
			Delete: connection.NewDelete(deps),
		},
		Reads: windowconnection.Reads{
			List: connection.NewList(deps), ListTypes: connection.NewListTypes(deps, cmp.Or[connectionport.OperationTitles](c.settings.Views.Operations, none.OperationTitles{})),
			RowOptions: openRowOptions(c, deps),
		},
	}
}

func openModelCalls(c pluginsWiring) connectionport.ModelCalls {
	return getForwardedCalls(c.settings.Views.Types, c.Endpoints.Endpoint)
}

func setModelCalls(c pluginsWiring, notices connectionport.ReportSetAside) {
	deps, ok := openConnectionDeps(c)
	if !ok {
		return
	}
	calls := modelcall.NewCalls(
		modelcall.Connections{
			Resolve: connection.NewResolve(deps).Execute, ReadSecrets: connection.NewReadSecrets(deps).Execute,
			Rows:   connection.NewListRows(deps, openConnectionRows(c)).Execute,
			Titles: c.settings.Views.Operations,
		},
		openCallSource(c, deps), c.vault.Clock, c.vault.Clock.Wait, notices,
	)
	lists := openRowLists(c)
	c.Endpoints.Lender.Update(func(l *hostendpoint.Lent) {
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

func openRowLists(c pluginsWiring) connectioncall.Rows {
	deps, ok := openConnectionDeps(c)
	if !ok {
		return nil
	}
	resolve := connection.NewResolve(deps).Execute
	options := openRowOptions(c, deps)
	if options == nil {
		return nil
	}
	return modelcall.NewRows(resolve, connection.NewListRows(deps, openConnectionRows(c)).Execute, options.ListOptions)
}

// GetConnectionRows is the listing of the rows plugin views offer a choice of, for a window that serves
// ConnectionCallService without the routes of the editor. It is nil where the installation runs no plugins.
func GetConnectionRows(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources) connectioncall.Rows {
	return openRowLists(plugins(c, exec, prefs, endpoints, res))
}

func openRowOptions(c pluginsWiring, deps connection.Deps) *rowoptions.Options {
	if c.Endpoints.RowOptions == nil || c.Endpoints.RunningProviders == nil {
		return nil
	}
	return rowoptions.NewOptions(rowoptions.Deps{
		Resolve: connection.NewResolve(deps).Execute, ReadSecrets: connection.NewReadSecrets(deps).Execute,
		Source: providerbridge.NewQuestionSource(c.Execution.Runtime.Activator), Running: c.Endpoints.RunningProviders,
		Lists: c.Endpoints.RowOptions, Titles: c.settings.Views.Operations,
		Grants: connection.NewGetGrantRevision(deps).Execute,
	})
}

func openCallSource(c pluginsWiring, deps connection.Deps) connectionport.ProviderSource {
	var source connectionport.ProviderSource = providerbridge.NewSource(c.Execution.Runtime.Activator)
	if c.Endpoints.Detection != nil {
		source = providerbridge.NewRedetectingSource(source, c.Endpoints.Detection)
	}
	if options := openRowOptions(c, deps); options != nil {
		source = providerbridge.NewCallReportingSource(source, options.RefreshRunning)
	}
	return source
}
