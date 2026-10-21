package service

import (
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// Mount registers the plugin's services on s.
func Mount(s *server.Server) error {
	return mount(s, &agentServer{})
}

func mount(s *server.Server, agent *agentServer) error {
	s.Register(numenv1connect.NewAgentServiceHandler(agent, s.HandlerOptions()...))
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
