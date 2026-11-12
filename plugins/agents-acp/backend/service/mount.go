package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/process"
)

// Mount registers the plugin's services on s.
func Mount(s *server.Server) error {
	agent, err := newAgentServer(process.LookPath, process.Launcher{}, clock.System{}, true)
	if err != nil {
		return err
	}
	return mount(s, agent)
}

func mount(s *server.Server, agent *agentServer) error {
	s.SetUpdateConfig(func(values map[string]string) {
		agent.registry.SetConfig(values)
		agent.settings.Apply(values)
	})
	s.Register(agentrun.NewHandler(agent, agent, s.HandlerOptions()...))
	connection.Mount(s, agent.Provider())
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
