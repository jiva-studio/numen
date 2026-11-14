package service

import (
	"errors"

	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/internal/adapter/antigravity"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/internal/adapter/mcpendpoint"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/completion"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/task"
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
	tools   task.Tools
	isAsync bool
}

func newAgent(p parts) (*agent, error) {
	values := new(settings.Settings)
	cli := antigravity.New(process.Supervisor{})
	models := catalog.New(cli, p.isAsync)
	return buildAgent(values, models, task.Dependencies{
		Settings: values, Program: cli, Models: models, Launcher: process.Launcher{},
		Tools: p.tools, ToolServer: mcpendpoint.Server{},
	})
}

func buildAgent(values *settings.Settings, models *catalog.Catalog, deps task.Dependencies) (*agent, error) {
	tasks, taskErr := task.New(deps)
	answers, compErr := completion.New(completion.Dependencies{Settings: values, Answerer: tasks, Models: models})
	if err := errors.Join(taskErr, compErr); err != nil {
		return nil, err
	}
	return &agent{settings: values, tasks: tasks, completion: answers}, nil
}
