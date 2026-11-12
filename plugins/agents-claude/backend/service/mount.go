package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/environment"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/hostenv"
)

// Mount registers the agent's services on s and hands Configure's settings to the agent.
// The runs the agent started end when s closes.
func Mount(s *server.Server) error {
	built, err := newAgent(parts{environment: hostenv.Process{}, clock: clock.System{}, isAsync: true})
	if err != nil {
		return err
	}
	return mount(s, built)
}

func mount(s *server.Server, a *agent) error {
	s.SetUpdateConfig(func(values map[string]string) { a.settings.Apply(values) })
	s.Register(agentrun.NewHandler(a, a, s.HandlerOptions()...))
	connection.Mount(s, a.Provider())
	s.OnClose(a.run.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		built, err := newAgent(parts{
			environment: hostenv.Fixed(environment.FromMap(vars)), clock: clock.System{}, isAsync: true,
		})
		if err != nil {
			return err
		}
		return mount(s, built)
	})
}
