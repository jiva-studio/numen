package plugin_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/toolbridge"
)

func lendEndpoint(host *container.PluginRoles, address string) (*hostendpoint.Services, hostendpoint.Callers) {
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
			return toolbridge.NewCatalog(plugin.NewToolHost(host.Activation, host.Registry))
		},
	}
	host.Services.SetHostEndpoint(hostendpoint.NewServer(address, callers))
	return services, callers
}

func lentCallers(host *container.PluginRoles) hostendpoint.Callers {
	_, callers := lendEndpoint(host, hostendpoint.Address)
	return callers
}
