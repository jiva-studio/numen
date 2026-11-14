package plugin_test

import (
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/toolbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func lendEndpoint(host *container.PluginRoles, address string) (*hostendpoint.Lender, hostendpoint.Callers) {
	lender := hostendpoint.NewLender()
	services := hostendpoint.NewServices(lender.Get)
	callers := hostendpoint.Callers{
		Identity: host.Activation, Leases: host.Activation,
		Lent: services,
		Tools: func() hostendpoint.PluginTools {
			toolHost := plugin.NewToolHost(host.Activation, host.Registry)
			return toolbridge.NewCatalog(toolHost, toolHost, host.Storage, testsupport.NewClock(), io.Discard)
		},
	}
	host.Endpoint.Replace(hostendpoint.NewServer(address, callers))
	return lender, callers
}

func lentCallers(host *container.PluginRoles) hostendpoint.Callers {
	_, callers := lendEndpoint(host, hostendpoint.Address)
	return callers
}
