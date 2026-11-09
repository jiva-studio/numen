// Package service is the backend of the Numen Agent plugin: numen's own agent, mounted on a plugin server.
package service

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// Mount registers the agent's services on s, asking the core of the host that the environment names
// for the turns of a conversation. It hands the settings Configure receives to the agent. The agent's
// tasks end when s closes.
func Mount(s *server.Server) error {
	client, err := connection.NewClientFromEnv()
	if err != nil {
		return err
	}
	mount(s, client)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. It asks the host those variables name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	client, err := connection.NewClientFromVars(vars)
	if err != nil {
		return nil, err
	}
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		mount(s, client)
		return nil
	})
}

func mount(s *server.Server, client *connection.Client) {
	agent := &Server{Unimplemented: agentrun.Unimplemented{}, chat: func(ctx context.Context, request connection.ChatRequest) (chatStream, error) {
		stream, err := client.Chat(ctx, request)
		if err != nil {
			return nil, err
		}
		return stream, nil
	}}
	s.SetUpdateConfig(func(values map[string]string, _ server.Secrets) { agent.Configure(values) })
	s.Register(agentrun.NewHandler(agent, s.HandlerOptions()...))
	s.OnClose(agent.Close)
}
