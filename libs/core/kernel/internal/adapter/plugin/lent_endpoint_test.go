package plugin_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agenttoolsbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
)

func lendEndpoint(host *pluginhost.Host, address string) (*hostendpoint.Services, hostendpoint.Callers) {
	services := hostendpoint.NewServices()
	callers := hostendpoint.Callers{
		Identity: host.Activation, Leases: host.Activation,
		Lent: services,
		Dependencies: func() hostendpoint.Dependencies {
			if registry := host.Services.GetDependencies(); registry != nil {
				return registry
			}
			return nil
		},
		Tools: func() hostendpoint.PluginTools {
			return agenttoolsbridge.NewCatalog(plugin.NewToolHost(host.Activation, host.Registry), agenttoolsbridge.Full)
		},
	}
	host.Services.SetHostEndpoint(hostendpoint.NewServer(address, callers))
	return services, callers
}

func lentCallers(host *pluginhost.Host) hostendpoint.Callers {
	_, callers := lendEndpoint(host, hostendpoint.Address)
	return callers
}
