// Package service is the backend of the Numen Agent plugin: numen's own agent, mounted on a plugin server.
package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// Mount registers the agent's services on s and hands the settings and secrets Configure receives to
// the agent. The agent's tasks end when s closes.
func Mount(s *server.Server) error {
	runner, err := newAgent()
	if err != nil {
		return err
	}
	s.SetUpdateConfig(func(values map[string]string, secrets server.Secrets) {
		runner.configure(values)
		runner.applySecrets(secrets)
	})
	s.Register(agentrun.NewHandler(runner, runner, s.HandlerOptions()...))
	s.OnClose(runner.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
