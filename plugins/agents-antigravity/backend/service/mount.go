package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/task"
)

// Mount registers the plugin's services on s, with the tools of the host the process environment
// names. The runs the plugin started end when s closes.
func Mount(s *server.Server) error {
	var host task.Tools
	if endpoint, err := hostcall.FromEnv(); err == nil {
		host = tools.NewClient(endpoint)
	}
	built, err := newAgent(parts{tools: host, isAsync: true})
	if err != nil {
		return err
	}
	s.SetUpdateConfig(built.settings.Apply)
	connection.Mount(s, built.Provider())
	s.OnClose(built.tasks.Close)
	return nil
}
