// Package agentmount mounts an agent service on the server of a plugin backend.
package agentmount

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// Agent is the agent service of a plugin with the hooks the server calls.
type Agent interface {
	agentrun.AgentServiceHandler
	// ApplyConfig receives the configuration the host sends.
	ApplyConfig(values map[string]string)
	// Close stops the work of the agent when the server closes.
	Close() error
}

// Mount registers the agent as the agent service of s.
func Mount(s *server.Server, agent Agent) {
	s.OnClose(agent.Close)
	s.SetUpdateConfig(agent.ApplyConfig)
	s.Register(agentrun.NewAgentServiceHandler(agent, s.HandlerOptions()...))
}

// Entry holds the constructor of an agent and provides the entry points of a plugin backend whose
// server serves that agent. Each mount creates its own agent.
type Entry struct {
	create func() Agent
}

// NewEntry returns the entry of a plugin backend that serves the agent create returns.
func NewEntry(create func() Agent) Entry {
	return Entry{create: create}
}

// Mount registers a new agent on s. It is the mount function of a launched process.
func (e Entry) Mount(s *server.Server) error {
	Mount(s, e.create())
	return nil
}

// NewHandler returns the handler of a host that links the plugin in.
func (e Entry) NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, e.Mount)
}
