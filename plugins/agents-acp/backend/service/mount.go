package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/ticker"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/usecase/task"
)

// idleCheck is how often the sessions that have gone unused are looked for.
const idleCheck = time.Minute

// Mount registers the plugin's services on s, with the tools of the host the process environment names.
func Mount(s *server.Server) error {
	return mountWith(s, hostcall.FromEnv)
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mountWith(s, func() (hostcall.Endpoint, error) { return hostcall.FromVars(vars) })
	})
}

func mountWith(s *server.Server, findHost func() (hostcall.Endpoint, error)) error {
	var host task.Tools
	if endpoint, err := findHost(); err == nil {
		host = tools.NewClient(endpoint)
	}
	agent, err := newAgentServer(machine{
		lookPath: process.LookPath, launcher: process.Launcher{}, tools: host, clock: clock.System{}, isAsync: true,
	})
	if err != nil {
		return err
	}
	return mount(s, agent)
}

func mount(s *server.Server, agent *agentServer) error {
	s.SetUpdateConfig(agent.settings.Apply)
	connection.Mount(s, agent.Provider())
	stopChecks := ticker.Start(idleCheck, func() error { return agent.tasks.CloseIdle(context.Background()) })
	s.OnClose(func() error {
		return errors.Join(stopChecks(), agent.tasks.Close(context.Background()))
	})
	return nil
}
