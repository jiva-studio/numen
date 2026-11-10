package plugin

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agenttoolsbridge"
)

func lendEndpoint(host *Host, address string) (*hostendpoint.Services, hostendpoint.Callers) {
	services := hostendpoint.NewServices()
	callers := hostendpoint.Callers{
		Identity: host.Activation, Leases: host.Activation,
		Completion: services, Vault: services, Texts: services, Search: services, Window: services, Work: services,
		Dependencies: hostendpoint.DependencySourceFunc(func() hostendpoint.Dependencies {
			if registry := host.Services.GetDependencies(); registry != nil {
				return registry
			}
			return nil
		}),
		Tools: hostendpoint.ToolSourceFunc(func() hostendpoint.PluginTools {
			return agenttoolsbridge.NewCatalog(NewToolHost(host.Activation, host.Registry), agenttoolsbridge.Full)
		}),
	}
	host.Services.SetHostEndpoint(hostendpoint.NewServer(address, callers))
	return services, callers
}

func lentCallers(host *Host) hostendpoint.Callers {
	_, callers := lendEndpoint(host, hostendpoint.Address)
	return callers
}
