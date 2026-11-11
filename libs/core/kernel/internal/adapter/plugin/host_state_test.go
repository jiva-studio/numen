package plugin

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func setPlugin(host *Host, p manifest.Discovered) {
	sups := host.Registry.Supervisors()
	host.Registry.Set().Put(p, sups.Register(p))
}

func getRawPlugin(host *Host, id string) manifest.Discovered {
	p, _ := host.Registry.Set().GetKnown(id)
	return p
}

func supervisorOf(host *Host, id string) *lifecycle.Supervisor {
	sup, _ := host.Registry.Supervisors().Get(id)
	return sup
}

func putSupervisor(host *Host, id string, sup *lifecycle.Supervisor) {
	host.Registry.Supervisors().Put(id, sup)
}

func endpointOf(host *Host, id string) transport.Endpoint {
	endpoint, _ := host.Registry.Supervisors().GetEndpoint(id)
	return endpoint
}

func putEndpoint(host *Host, id string, endpoint transport.Endpoint) {
	host.Registry.Supervisors().SetEndpoint(id, endpoint)
}

func dropEndpoint(host *Host, id string) {
	host.Registry.Supervisors().Drop(id)
}
