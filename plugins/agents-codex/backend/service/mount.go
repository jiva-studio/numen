package service

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentmount"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
)

// Mount registers the plugin's services on a server. NewHandler returns the handler of the plugin's
// backend for a host that links the plugin in, made from the variables a launched process of the
// plugin is given.
var Mount, NewHandler = agentmount.NewEntry(newAgent)

func newAgent() agentmount.Agent {
	agent := new(agentServer)
	agent.codexModels.isAsync = true
	agent.codexModels.clock = clock.System{}
	return agent
}
