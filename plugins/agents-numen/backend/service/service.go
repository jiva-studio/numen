// Package service is the backend of the Numen Agent plugin: numen's own agent, mounted on a plugin server
// as the provider of the connection type numen.
package service

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/hostchat"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/hosttools"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/answer"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/models"
)

// Mount registers acp.task and llm.models on s and hands the settings Configure receives to the agent.
// The agent's tasks end when s closes. The models, the connections that serve them and the tools are
// those of the host the environment names.
func Mount(s *server.Server) error {
	chat, connections, err := hostchat.FromEnvironment()
	if err != nil {
		return err
	}
	tools, err := hosttools.FromEnvironment()
	if err != nil {
		return err
	}
	return mount(s, host{chat: chat, models: connections, tools: tools})
}

// host is what the agent reaches through the core: the model of a turn, the connections that serve
// llm.chat with their models, and the tools.
type host struct {
	chat   ports.Chat
	models ports.Models
	tools  ports.Tools
}

func mount(s *server.Server, reach host) error {
	agent, err := newAgent(reach)
	if err != nil {
		return err
	}
	s.SetUpdateConfig(agent.settings.Configure)
	connection.Mount(s, agent.provider)
	s.OnClose(agent.answerer.Close)
	return nil
}

// agent is the provider of acp.task and llm.models, the settings it reads and the answerer that carries
// out its tasks.
type agent struct {
	settings *settings.Store
	answerer *answer.Answerer
	provider connection.Provider
}

func newAgent(reach host) (*agent, error) {
	store := settings.NewStore()
	answerer, err := answer.New(answer.Parts{Store: store, Chat: reach.chat, Models: reach.models, Tools: reach.tools, Clock: clock.System{}})
	if err != nil {
		return nil, err
	}
	lister, err := models.NewLister(reach.models)
	if err != nil {
		return nil, err
	}
	provider := llm.Provide(acp.Provide(nil, answerer), llm.Roles{Models: modelOptions{lister: lister}})
	return &agent{settings: store, answerer: answerer, provider: provider}, nil
}

// modelOptions serves llm.models for a connection of the type numen: the models of every connection that
// serves llm.chat.
type modelOptions struct {
	lister *models.Lister
}

func (m modelOptions) ListModels(ctx context.Context, _ connection.Call, _ connection.OptionsQuery) (connection.ListedOptions, error) {
	listing, err := m.lister.ListOptions(ctx)
	if err != nil {
		return connection.ListedOptions{}, err
	}
	options := make([]connection.Option, len(listing.Options))
	for i, one := range listing.Options {
		options[i] = connection.Option{Value: one.Value, Title: one.Title, Group: ""}
	}
	return connection.ListedOptions{Options: options, IsPartial: listing.IsPartial}, nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made from
// the variables a launched process of the plugin is given. The models, the connections that serve them
// and the tools are those of the host those variables name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	chat, connections, err := hostchat.FromVariables(vars)
	if err != nil {
		return nil, err
	}
	tools, err := hosttools.FromVariables(vars)
	if err != nil {
		return nil, err
	}
	reach := host{chat: chat, models: connections, tools: tools}
	return server.NewLinkedHandler(vars, func(s *server.Server) error { return mount(s, reach) })
}
