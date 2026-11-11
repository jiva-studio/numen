package plugin

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func setPlugin(host *Host, id string, p manifest.Discovered) {
	host.Registry.set.mu.Lock()
	defer host.Registry.set.mu.Unlock()
	host.Registry.set.plugins[id] = p
}

func getRawPlugin(host *Host, id string) manifest.Discovered {
	p, _ := host.Registry.set.getKnown(id)
	return p
}

func supervisorOf(host *Host, id string) *lifecycle.Supervisor {
	sup, _ := host.Registry.sups.Get(id)
	return sup
}

func putSupervisor(host *Host, id string, sup *lifecycle.Supervisor) {
	host.Registry.sups.Put(id, sup)
}

func endpointOf(host *Host, id string) transport.Endpoint {
	endpoint, _ := host.Registry.sups.GetEndpoint(id)
	return endpoint
}

func putEndpoint(host *Host, id string, endpoint transport.Endpoint) {
	host.Registry.sups.SetEndpoint(id, endpoint)
}

func dropEndpoint(host *Host, id string) {
	host.Registry.sups.Drop(id)
}
