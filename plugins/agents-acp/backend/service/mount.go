package service

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// Mount registers the plugin's services on s.
func Mount(s *server.Server) error {
	return mount(s, &agentServer{})
}

func mount(s *server.Server, agent *agentServer) error {
	s.SetUpdateConfig(agent.applyConfig)
	s.Register(agentrun.NewAgentServiceHandler(agent, s.HandlerOptions()...))
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
