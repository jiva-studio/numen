package container

import (
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	windowconnection "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/rowoptions"
)

func hasPluginHost(c pluginsWiring) bool {
	return c.PluginHost.Views.Types != nil
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
		Refresh:     c.PluginHost.Configuration.Refresh,
		Report:      s.ErrorHandler,
	})
}

func openConnectionDeps(c pluginsWiring) (connection.Deps, bool) {
	if !hasPluginHost(c) {
		return connection.Deps{}, false
	}
	deps := connection.NewDeps(openConnectionRepository(c.settings), c.PluginHost.Views.Types, c.KeyStore, openConnectionRows(c)).
		AddDetection(openDetection(c))
	if options := openRowOptions(c, deps); options != nil {
		deps = deps.AddOptionLists(options)
	}
	return deps, true
}

func openDetection(c pluginsWiring) connectionport.Detected {
	detection := c.PluginHost.Detection
	if detection == nil {
		source := providerbridge.NewQuestionSource(c.PluginHost.Runtime)
		return providerbridge.NewDetection(c.PluginHost.Views.Types, source, clock.System{})
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
			List: connection.NewList(deps), ListTypes: connection.NewListTypes(deps, c.PluginHost.Views.Operations),
			RowOptions: openRowOptions(c, deps),
		},
	}
}

func openModelCalls(c pluginsWiring) connectionport.ModelCalls {
	if !hasPluginHost(c) {
		return nil
	}
	return modelcall.NewForwardedCalls(func() connectionport.ModelCalls { return c.PluginHost.Endpoint.GetModelCalls() })
}

func setModelCalls(c pluginsWiring, notices connectionport.ReportSetAside) {
	deps, ok := openConnectionDeps(c)
	if !ok {
		return
	}
	c.PluginHost.Endpoint.SetModelCalls(modelcall.NewCalls(
		modelcall.Connections{
			Resolve: connection.NewResolve(deps).Execute, ReadSecrets: connection.NewReadSecrets(deps).Execute,
			Rows:   connection.NewListRows(deps, openConnectionRows(c)).Execute,
			Titles: c.PluginHost.Views.Operations,
		},
		openCallSource(c, deps), clock.System{}, clock.System{}.Wait, notices,
	))
	if lists := openRowLists(c); lists != nil {
		c.PluginHost.Endpoint.SetRowLists(lists)
	}
}

// GetConnectionCalls is the calls plugin views make through their connections, for a window that serves
// ConnectionCallService without the routes of the editor. It is nil where the installation runs no
// plugins.
func GetConnectionCalls(c Config) connectionport.ModelCalls {
	return openModelCalls(plugins(c))
}

func openRowLists(c pluginsWiring) connectionport.RowLists {
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
func GetConnectionRows(c Config) connectionport.RowLists {
	return openRowLists(plugins(c))
}

func openRowOptions(c pluginsWiring, deps connection.Deps) *rowoptions.Options {
	if c.PluginHost.RowOptions == nil || c.PluginHost.RunningProviders == nil {
		return nil
	}
	options := rowoptions.NewOptions(
		connection.NewResolve(deps).Execute, connection.NewReadSecrets(deps).Execute,
		providerbridge.NewQuestionSource(c.PluginHost.Runtime), c.PluginHost.RunningProviders, c.PluginHost.RowOptions,
	)
	options.Titles = c.PluginHost.Views.Operations
	return options
}

// openCallSource opens the providers of the calls through connections. A call a provider answered makes
// the option lists of its connection refresh while the plugin runs.
func openCallSource(c pluginsWiring, deps connection.Deps) connectionport.ProviderSource {
	var source connectionport.ProviderSource = providerbridge.NewSource(c.PluginHost.Runtime)
	if c.PluginHost.Detection != nil {
		source = providerbridge.NewRedetectingSource(source, c.PluginHost.Detection)
	}
	if options := openRowOptions(c, deps); options != nil {
		source = providerbridge.NewCallReportingSource(source, options.RefreshRunning)
	}
	return source
}
