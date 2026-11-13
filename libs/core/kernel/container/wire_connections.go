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
		Types:       c.PluginHost.Views.Types,
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
	return connection.NewDeps(openConnectionRepository(c.settings), c.PluginHost.Views.Types, c.KeyStore, openConnectionRows(c)), true
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
			RowOptions: rowoptions.NewOptions(connection.NewResolve(deps).Execute, connection.NewReadSecrets(deps).Execute, providerbridge.NewSource(c.PluginHost.Runtime)),
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
	resolve, readSecrets := connection.NewResolve(deps).Execute, connection.NewReadSecrets(deps).Execute
	source := providerbridge.NewSource(c.PluginHost.Runtime)
	rows := connection.NewListRows(deps, openConnectionRows(c)).Execute
	c.PluginHost.Endpoint.SetModelCalls(callsAndRows{
		Calls: modelcall.NewCalls(
			modelcall.Connections{Resolve: resolve, ReadSecrets: readSecrets, Rows: rows},
			source, clock.System{}, clock.System{}.Wait, notices,
		),
		Rows: modelcall.NewRows(resolve, rows, rowoptions.NewOptions(resolve, readSecrets, source).ListOptions),
	})
}

// callsAndRows are the calls of the plugins with the listing of their rows, which the views of the window
// reach through the plugin endpoint.
type callsAndRows struct {
	*modelcall.Calls
	*modelcall.Rows
}

// GetConnectionCalls is the calls plugin views make through their connections, for a window that serves
// ConnectionCallService without the routes of the editor. It is nil where the installation runs no
// plugins.
func GetConnectionCalls(c Config) connectionport.ModelCalls {
	return openModelCalls(plugins(c))
}
