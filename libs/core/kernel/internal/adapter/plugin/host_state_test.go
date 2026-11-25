package plugin_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func setPlugin(management *container.PluginManagement, id string, p manifest.DiscoveredPlugin) {
	plugin.SetPlugin(management.Registry, id, p)
}

func getRawPlugin(management *container.PluginManagement, id string) manifest.DiscoveredPlugin {
	return plugin.GetRawPlugin(management.Registry, id)
}

func supervisorOf(management *container.PluginManagement, id string) *lifecycle.Supervisor {
	sup, _ := plugin.GetSupervisors(management.Registry).Get(id)
	return sup
}

func putSupervisor(management *container.PluginManagement, id string, sup *lifecycle.Supervisor) {
	plugin.GetSupervisors(management.Registry).Put(id, sup)
}

func endpointOf(management *container.PluginManagement, id string) transport.Endpoint {
	endpoint, _ := plugin.GetSupervisors(management.Registry).GetEndpoint(id)
	return endpoint
}

func putEndpoint(management *container.PluginManagement, id string, endpoint transport.Endpoint) {
	plugin.GetSupervisors(management.Registry).SetEndpoint(id, endpoint)
}

func dropEndpoint(management *container.PluginManagement, id string) {
	plugin.GetSupervisors(management.Registry).Drop(id)
}
