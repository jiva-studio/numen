package service

import (
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// Mount registers the plugin's services on s.
func Mount(s *server.Server) error {
	agent := new(agentServer)
	agent.models = newModelCache(true)
	s.OnClose(func() {
		if err := agent.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close the agent: %v\n", err)
		}
	})
	s.SetUpdateConfig(agent.applyConfig)
	s.Register(agentrun.NewAgentServiceHandler(agent, s.HandlerOptions()...))
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
