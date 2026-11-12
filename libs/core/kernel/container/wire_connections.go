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
		Create: connection.NewCreate(deps, settingsadapter.CreateConnectionID), Update: connection.NewUpdate(deps),
		Delete: connection.NewDelete(deps), List: connection.NewList(deps), ListTypes: connection.NewListTypes(deps),
	}
}

func openModelLists(c pluginsWiring) *modellist.Lists {
	deps, ok := openConnectionDeps(c)
	if !ok {
		return nil
	}
	return modellist.NewLists(connection.NewResolve(deps).Execute, connection.NewReadSecrets(deps).Execute, providerbridge.NewSource(c.PluginHost.Runtime))
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
	rows := openConnectionRows(c)
	c.PluginHost.Endpoint.SetModelCalls(modelcall.NewCalls(
		modelcall.Connections{
			Resolve: connection.NewResolve(deps).Execute, ReadSecrets: connection.NewReadSecrets(deps).Execute, Rows: rows,
		},
		providerbridge.NewSource(c.PluginHost.Runtime), clock.System{}, clock.System{}.Wait, notices,
	))
}
