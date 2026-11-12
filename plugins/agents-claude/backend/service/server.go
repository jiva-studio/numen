package service

import (
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/executable"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/mcpendpoint"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/toolconfig"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/usecase/completion"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/usecase/task"
)

// agent holds the use cases of the plugin.
type agent struct {
	settings   *settings.Settings
	tasks      *task.Runner
	completion *completion.Completion
}

// parts are what differs between the plugin process, a host that links the plugin in, and a test.
// Tools is nil where the plugin has no host to list the tools on.
type parts struct {
	environment ports.Environment
	tools       task.Tools
	isAsync     bool
}

func newAgent(p parts) (*agent, error) {
	values := new(settings.Settings)
	lister := process.Lister{}
	models, err := catalog.New(lister, lister.ReportError, p.isAsync)
	if err != nil {
		return nil, err
	}
	tasks, err := task.New(task.Dependencies{
		Settings: values, Finder: executable.Finder{}, Reasoning: models, Launcher: process.Launcher{},
		ToolConfig: toolconfig.Files{}, Environment: p.environment, Tools: p.tools, ToolServer: mcpendpoint.Server{},
	})
	if err != nil {
		return nil, err
	}
	answers, err := completion.New(completion.Dependencies{Settings: values, Answerer: tasks, Models: models})
	if err != nil {
		return nil, err
	}
	return &agent{settings: values, tasks: tasks, completion: answers}, nil
}
