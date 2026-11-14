package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/hostenv"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/usecase/task"
)

// Mount registers the plugin's services on s, with the tools of the host the process environment
// names. The runs the plugin started end when s closes.
func Mount(s *server.Server) error {
	var host task.Tools
	if endpoint, err := hostcall.FromEnv(); err == nil {
		host = tools.NewClient(endpoint)
	}
	built, err := newAgent(parts{environment: hostenv.Process{}, tools: host, isAsync: true})
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
