package plugin

import (
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// openedHosts keeps the parts the roles of each host openHost made share.
var openedHosts sync.Map

// openHost makes a host as NewHost does, and keeps its shared parts for the helpers of the tests.
func openHost(clock machineport.Clock, target string, dirs ...string) *Host {
	host, made := assemble(clock, target, dirs)
	openedHosts.Store(host, made)
	return host
}

func partsOf(host *Host) assembly {
	made, _ := openedHosts.Load(host)
	return made.(assembly)
}

func setPlugin(host *Host, p manifest.Discovered) {
	parts := partsOf(host)
	parts.set.Put(p, parts.sups.Register(p))
}

func getRawPlugin(host *Host, id string) manifest.Discovered {
	p, _ := partsOf(host).set.GetKnown(id)
	return p
}

func supervisorOf(host *Host, id string) *lifecycle.Supervisor {
	sup, _ := partsOf(host).sups.Get(id)
	return sup
}

func putSupervisor(host *Host, id string, sup *lifecycle.Supervisor) {
	partsOf(host).sups.Put(id, sup)
}

func endpointOf(host *Host, id string) transport.Endpoint {
	endpoint, _ := partsOf(host).sups.GetEndpoint(id)
	return endpoint
}

func putEndpoint(host *Host, id string, endpoint transport.Endpoint) {
	partsOf(host).sups.SetEndpoint(id, endpoint)
}

func dropEndpoint(host *Host, id string) {
	partsOf(host).sups.Drop(id)
}

func evictorOf(host *Host) *supervision.Evictor {
	return partsOf(host).evictor
}

func stopperOf(host *Host) *supervision.Stopper {
	return partsOf(host).stopper
}
