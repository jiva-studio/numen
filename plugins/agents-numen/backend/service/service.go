// Package service is the backend of the Numen Agent plugin: numen's own agent, mounted on a plugin server
// as the provider of the connection type numen.
package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/hostchat"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/hosttools"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/answer"
)

// Mount registers acp.task on s and hands the settings Configure receives to the agent. The agent's tasks
// end when s closes. The model API and the tools are those of the host the environment names.
func Mount(s *server.Server) error {
	chat, err := hostchat.FromEnvironment()
	if err != nil {
		return err
	}
	tools, err := hosttools.FromEnvironment()
	if err != nil {
		return err
	}
	return mount(s, chat, tools)
}

func mount(s *server.Server, chat ports.Chat, tools ports.Tools) error {
	agent, err := newAgent(chat, tools)
	if err != nil {
		return err
	}
	s.SetUpdateConfig(agent.settings.Configure)
	connection.Mount(s, agent.provider)
	s.OnClose(agent.answerer.Close)
	return nil
}

// agent is the provider of acp.task, the settings it reads and the answerer that carries out its tasks.
type agent struct {
	settings *settings.Store
	answerer *answer.Answerer
	provider connection.Provider
}

func newAgent(chat ports.Chat, tools ports.Tools) (*agent, error) {
	store := settings.NewStore()
	answerer, err := answer.New(store, chat, tools, clock.System{})
	if err != nil {
		return nil, err
	}
	return &agent{settings: store, answerer: answerer, provider: acp.Provide(nil, answerer)}, nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made from
// the variables a launched process of the plugin is given. The model API and the tools are those of the
// host those variables name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	chat, err := hostchat.FromVariables(vars)
	if err != nil {
		return nil, err
	}
	tools, err := hosttools.FromVariables(vars)
	if err != nil {
		return nil, err
	}
	return server.NewLinkedHandler(vars, func(s *server.Server) error { return mount(s, chat, tools) })
}
