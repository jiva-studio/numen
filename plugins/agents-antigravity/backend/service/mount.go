package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// Mount registers the plugin's services on s.
func Mount(s *server.Server) error {
	agent := newAgentServer(true)
	s.OnClose(agent.Close)
	s.SetUpdateConfig(func(values map[string]string, _ server.Secrets) { agent.settings.Apply(values) })
	s.Register(agentrun.NewHandler(agent, agent, s.HandlerOptions()...))
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
