// Package service is the backend of the Numen Agent plugin: numen's own agent, mounted on a plugin server.
package service

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"net/http"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// Mount registers the agent's services on s and hands the settings and secrets Configure receives to
// the agent. The agent's tasks end when s closes.
func Mount(s *server.Server) error {
	agent := &Server{
		UnimplementedAgentServiceHandler: agentrun.UnimplementedAgentServiceHandler{},
		Client:                           nil,
		mu:                               sync.Mutex{},
		baseURL:                          "",
		key:                              "",
		model:                            "",
		temperature:                      nil,
		maxTokens:                        nil,
		conversations:                    nil,
		models:                           nil,
		isListed:                         false,
		registry:                         agentrun.Registry{},
	}
	s.SetUpdateConfig(agent.Configure)
	s.Register(agentrun.NewAgentServiceHandler(agent, s.HandlerOptions()...))
	s.OnClose(agent.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
