package plugin_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func setPlugin(host *container.PluginRoles, id string, p manifest.Discovered) {
	plugin.SetPlugin(host.Registry, id, p)
}

func getRawPlugin(host *container.PluginRoles, id string) manifest.Discovered {
	return plugin.GetRawPlugin(host.Registry, id)
}

func supervisorOf(host *container.PluginRoles, id string) *lifecycle.Supervisor {
	sup, _ := plugin.GetSupervisors(host.Registry).Get(id)
	return sup
}

func putSupervisor(host *container.PluginRoles, id string, sup *lifecycle.Supervisor) {
	plugin.GetSupervisors(host.Registry).Put(id, sup)
}

func endpointOf(host *container.PluginRoles, id string) transport.Endpoint {
	endpoint, _ := plugin.GetSupervisors(host.Registry).GetEndpoint(id)
	return endpoint
}

func putEndpoint(host *container.PluginRoles, id string, endpoint transport.Endpoint) {
	plugin.GetSupervisors(host.Registry).SetEndpoint(id, endpoint)
}

func dropEndpoint(host *container.PluginRoles, id string) {
	plugin.GetSupervisors(host.Registry).Drop(id)
}
