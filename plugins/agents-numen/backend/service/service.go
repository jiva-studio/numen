// Package service is the backend of the Numen Agent plugin: numen's own agent, mounted on a plugin server.
package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/hostchat"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// Mount registers the agent's services on s and hands the settings Configure receives to the agent.
// The agent's tasks end when s closes. The model API of the host is the one the environment names.
func Mount(s *server.Server) error {
	chat, err := hostchat.FromEnvironment()
	if err != nil {
		return err
	}
	return mount(s, chat)
}

func mount(s *server.Server, chat ports.Chat) error {
	runner, err := newAgent(chat)
	if err != nil {
		return err
	}
	s.SetUpdateConfig(runner.settings.Configure)
	s.Register(agentrun.NewHandler(runner, runner, s.HandlerOptions()...))
	s.OnClose(runner.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. The model API of the host is the one
// those variables name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	chat, err := hostchat.FromVariables(vars)
	if err != nil {
		return nil, err
	}
	return server.NewLinkedHandler(vars, func(s *server.Server) error { return mount(s, chat) })
}
