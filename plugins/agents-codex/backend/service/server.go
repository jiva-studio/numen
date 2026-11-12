package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/codexcli"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/mcpendpoint"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/completion"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/modellist"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/task"
)

// agentServer holds the use cases of the plugin.
type agentServer struct {
	settings   *settings.Settings
	tasks      *task.Runner
	completion *completion.Completion
}

// parts are what differs between the plugin process, a host that links the plugin in, and a test.
// Tools is nil where the plugin has no host to list the tools on.
type parts struct {
	tools   task.Tools
	isAsync bool
}

func newAgentServer(p parts) (*agentServer, error) {
	values := new(settings.Settings)
	tasks, err := task.New(task.Dependencies{
		Settings: values, Finder: codexcli.Installation{}, Commands: codexcli.Commands{}, Launcher: codexcli.Launcher{},
		Tools: p.tools, ToolServer: mcpendpoint.Server{},
	})
	if err != nil {
		return nil, err
	}
	models := modellist.NewCache(codexcli.Models{}, clock.System{}, p.isAsync)
	answers, err := completion.New(completion.Dependencies{Settings: values, Answerer: tasks, Models: models})
	if err != nil {
		return nil, err
	}
	return &agentServer{settings: values, tasks: tasks, completion: answers}, nil
}
