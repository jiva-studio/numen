package service

import (
	"context"
	"errors"
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
	var host task.Tools
	if endpoint, err := hostcall.ReadEnv(); err == nil {
		host = tools.NewClient(endpoint)
	}
	return mountMachine(s, machine{
		lookPath: process.LookPath, launcher: process.Launcher{}, tools: host, clock: clock.System{}, isAsync: true,
	})
}

func mountMachine(s *server.Server, on machine) error {
	agent, err := newAgentServer(on)
	if err != nil {
		return err
	}
	return mount(s, agent)
}

func mount(s *server.Server, agent *agentServer) error {
	s.SetUpdateConfig(agent.settings.Apply)
	connection.Mount(s, agent.Provider())
	stopChecks := ticker.Start(idleCheck, func() error { return agent.tasks.CloseIdle(context.Background()) })
	s.AddCloser(func() error {
		return errors.Join(stopChecks(), agent.tasks.Close(context.Background()))
	})
	return nil
}
