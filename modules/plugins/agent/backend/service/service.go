// Package service is the backend of the Numen Agent plugin: numen's own agent, the harness of the
// plugin SDK, mounted on a plugin server.
package service

import (
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/harness"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// Mount registers the agent's services on s and hands the settings and secrets Configure receives to
// the agent. The agent's tasks end when s closes.
func Mount(s *server.Server) error {
	agent := &harness.Server{}
	s.SetUpdateConfig(agent.Configure)
	s.Register(numenv1connect.NewAgentServiceHandler(agent, s.HandlerOptions()...))
	s.Register(numenv1connect.NewCompletionServiceHandler(agent, s.HandlerOptions()...))
	s.OnClose(func() { _ = agent.Close() })
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
