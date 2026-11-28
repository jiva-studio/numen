package plugin_test

import (
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/toolbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func lendEndpoint(host *pluginhost.Host, address string) (*hostendpoint.Lender, hostendpoint.Callers) {
	lender := hostendpoint.NewLender()
	services := hostendpoint.NewServices(lender.Get)
	callers := hostendpoint.Callers{
		Activation: host.Management.Activation,
		Lent:       services,
		GetTools: func() hostendpoint.PluginTools {
			toolHost := plugin.NewToolHost(host.Management.Activation, host.Management.Registry)
			return toolbridge.NewCatalog(toolHost, toolHost, host.Supervision.Storage, func() string { return "r1" }, testsupport.NewClock(), io.Discard)
		},
	}
	host.Supervision.Endpoint.Replace(hostendpoint.NewServer(address, callers, hostendpoint.AccountCalls{}))
	return lender, callers
}

func lentCallers(host *pluginhost.Host) hostendpoint.Callers {
	_, callers := lendEndpoint(host, hostendpoint.Address)
	return callers
}
