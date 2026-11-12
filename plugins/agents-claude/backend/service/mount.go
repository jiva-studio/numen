package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/environment"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/hostenv"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/usecase/task"
)

// Mount registers the plugin's services on s, with the tools of the host the process environment
// names. The runs the plugin started end when s closes.
func Mount(s *server.Server) error {
	return mountWith(s, hostenv.Process{}, hostcall.FromEnv)
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mountWith(s, hostenv.Fixed(environment.FromMap(vars)), func() (hostcall.Endpoint, error) { return hostcall.FromVars(vars) })
	})
}

func mountWith(s *server.Server, env ports.Environment, findHost func() (hostcall.Endpoint, error)) error {
	var host task.Tools
	if endpoint, err := findHost(); err == nil {
		host = tools.NewClient(endpoint)
	}
	built, err := newAgent(parts{environment: env, tools: host, isAsync: true})
	if err != nil {
		return err
	}
	return mount(s, built)
}

func mount(s *server.Server, a *agent) error {
	s.SetUpdateConfig(a.settings.Apply)
	connection.Mount(s, a.Provider())
	s.OnClose(a.tasks.Close)
	return nil
}
