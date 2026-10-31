package service

import "github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentmount"

// entry provides the mount function and the linked handler constructor of the plugin backend.
var entry = agentmount.NewEntry(newAgent)

// Mount registers the agent service on s.
var Mount = entry.Mount

// NewHandler returns the handler of a host that links the plugin in.
var NewHandler = entry.NewHandler

func newAgent() agentmount.Agent {
	agent := new(agentServer)
	agent.models = newModelCache(true)
	return agent
}
