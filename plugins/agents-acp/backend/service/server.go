package service

import (
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/mcpendpoint"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/usecase/completion"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/usecase/detection"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/usecase/listing"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/usecase/task"
)

// machine is what the plugin reaches on the machine and on the host. Tools is nil where the plugin has
// no host to list the tools on.
type machine struct {
	lookPath func(string) (string, error)
	launcher acp.Launcher
	tools    task.Tools
	clock    cache.Clock
	isAsync  bool
}

type agentServer struct {
	settings   *settings.Settings
	completion *completion.Completion
	tasks      *task.Runner
	detection  *detection.Detection
}

func newAgentServer(on machine) (*agentServer, error) {
	values := new(settings.Settings)
	lister, listErr := listing.New(on.launcher, on.clock, on.isAsync)
	answers, compErr := completion.New(completion.Dependencies{Settings: values, Launcher: on.launcher, Listings: lister, LookPath: on.lookPath})
	tasks, taskErr := task.New(task.Dependencies{
		Launcher: on.launcher, LookPath: on.lookPath, Tools: on.tools, ToolServer: mcpendpoint.Server{},
		Clock: on.clock, Settings: values,
	})
	if err := errors.Join(listErr, compErr, taskErr); err != nil {
		return nil, err
	}
	return &agentServer{settings: values, completion: answers, tasks: tasks, detection: detection.New(on.lookPath)}, nil
}
