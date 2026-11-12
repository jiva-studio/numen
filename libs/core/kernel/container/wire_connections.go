package container

import (
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	windowconnection "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modellist"
)

// hasPluginHost reports whether the installation runs a plugin host. Without one there is no kind of
// connection, and the connections, the lists of models and the calls on models are not wired.
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

// openConnectionRows returns the rows of the settings of the plugins, which are the connections each plugin
// may use, and the link between a connection and the settings that name it.
func openConnectionRows(c pluginsWiring) *settingsadapter.ConnectionRows {
	s := c.settings
	return settingsadapter.NewConnectionRows(settingsadapter.ConnectionRowsSource{
		Connections: openConnectionRepository(s),
		Plugins:     settingsPlugins(s),
		Declared:    getDeclarations(s),
		ReadStored:  settingsStore(s).ReadUncheckedFile,
		Write:       getSettingsWriter(s),
		Refresh:     c.PluginHost.Configuration.Refresh,
		Report:      s.ErrorHandler,
	})
}

// openConnectionDeps returns the parts the use cases on the connections share, or false where the installation runs
// no plugin host.
func openConnectionDeps(c pluginsWiring) (connection.Deps, bool) {
	if !hasPluginHost(c) {
		return connection.Deps{}, false
	}
	return connection.NewDeps(openConnectionRepository(c.settings), c.PluginHost.Views.Types, c.KeyStore, openConnectionRows(c)), true
}

// openConnectionUseCases returns the use cases a window works the connections with, or nil where the installation
// runs no plugin host.
func openConnectionUseCases(c pluginsWiring) *windowconnection.UseCases {
	deps, ok := openConnectionDeps(c)
	if !ok {
		return nil
	}
	return &windowconnection.UseCases{
		Create: connection.NewCreate(deps, settingsadapter.CreateConnectionID), Update: connection.NewUpdate(deps),
		Delete: connection.NewDelete(deps), List: connection.NewList(deps), ListTypes: connection.NewListTypes(deps),
	}
}

// openModelLists returns the models and the reasoning levels a provider gives, or nil where the installation
// runs no plugin host.
func openModelLists(c pluginsWiring) *modellist.Lists {
	deps, ok := openConnectionDeps(c)
	if !ok {
		return nil
	}
	return modellist.NewLists(connection.NewResolve(deps).Execute, connection.NewReadSecrets(deps).Execute, providerbridge.NewSource(c.PluginHost.Runtime))
}

// openModelCalls returns the calls on models that setModelCalls gives the plugin host, or nil where the
// installation runs no plugin host. A call made before setModelCalls finds no connection.
func openModelCalls(c pluginsWiring) connectionport.ModelCalls {
	if !hasPluginHost(c) {
		return nil
	}
	return modelcall.NewForwardedCalls(func() connectionport.ModelCalls { return c.PluginHost.Endpoint.GetModelCalls() })
}

// setModelCalls lets the plugins ask the host for the operations on models. notices tells the person of a
// connection set aside. Nothing is set where the installation runs no plugin host.
func setModelCalls(c pluginsWiring, notices connectionport.Notices) {
	deps, ok := openConnectionDeps(c)
	if !ok {
		return
	}
	rows := openConnectionRows(c)
	c.PluginHost.Endpoint.SetModelCalls(modelcall.NewCalls(
		connection.NewResolve(deps).Execute, connection.NewReadSecrets(deps).Execute, rows, providerbridge.NewSource(c.PluginHost.Runtime),
		clock.System{}, clock.System{}.Wait, notices,
	))
}
