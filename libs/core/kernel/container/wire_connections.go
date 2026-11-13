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
	deps := connection.NewDeps(openConnectionRepository(c.settings), c.PluginHost.Views.Types, c.KeyStore, openConnectionRows(c))
	return deps.AddDetection(openDetection(c)), true
}

func openDetection(c pluginsWiring) connectionport.Detected {
	detection := c.PluginHost.Detection
	if detection == nil {
		return providerbridge.NewDetection(c.PluginHost.Views.Types, providerbridge.NewSource(c.PluginHost.Runtime))
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
	c.PluginHost.Endpoint.SetModelCalls(modelcall.NewCalls(
		modelcall.Connections{
			Resolve: connection.NewResolve(deps).Execute, ReadSecrets: connection.NewReadSecrets(deps).Execute,
			Rows: connection.NewListRows(deps, openConnectionRows(c)).Execute,
		},
		providerbridge.NewSource(c.PluginHost.Runtime), clock.System{}, clock.System{}.Wait, notices,
	))
}

// GetConnectionCalls is the calls plugin views make through their connections, for a window that serves
// ConnectionCallService without the routes of the editor. It is nil where the installation runs no
// plugins.
func GetConnectionCalls(c Config) connectionport.ModelCalls {
	return openModelCalls(plugins(c))
}
