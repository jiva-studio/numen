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

// NewEntry returns the entry points of a plugin backend whose server serves the agent that create
// returns: the mount function of a launched process, and the handler constructor of a host that
// links the plugin in. Each mount creates its own agent.
func NewEntry(create func() Agent) (
	mount func(*server.Server) error,
	newHandler func(vars map[string]string) (http.Handler, error),
) {
	mount = func(s *server.Server) error {
		Mount(s, create())
		return nil
	}
	newHandler = func(vars map[string]string) (http.Handler, error) {
		return server.NewLinkedHandler(vars, mount)
	}
	return mount, newHandler
}
