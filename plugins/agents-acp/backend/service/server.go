package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/agents"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/usecase/ask"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/usecase/completion"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/usecase/programs"
)

type agentServer struct {
	agentrun.Unimplemented
	registry   *agents.Registry
	settings   *settings.Settings
	asker      *ask.Asker
	lister     *programs.Lister
	completion *completion.Completion
}

func newAgentServer(lookPath func(string) (string, error), launcher acp.Launcher, source cache.Clock, isAsync bool) (*agentServer, error) {
	registry := agents.NewRegistry(lookPath)
	asker, err := ask.New(registry, launcher)
	if err != nil {
		return nil, err
	}
	lister, err := programs.New(registry, launcher, source, isAsync)
	if err != nil {
		return nil, err
	}
	values := new(settings.Settings)
	answers, err := completion.New(completion.Dependencies{Settings: values, Answerer: asker, Listings: lister, LookPath: lookPath})
	if err != nil {
		return nil, err
	}
	return &agentServer{
		Unimplemented: agentrun.Unimplemented{}, registry: registry, settings: values, asker: asker, lister: lister, completion: answers,
	}, nil
}

// Ask runs the task on the ACP agent the task names.
func (s *agentServer) Ask(ctx context.Context, task *agentrun.Request, send func(*agentrun.Response) error) error {
	return s.asker.Execute(ctx, task, send)
}

// GetPrograms lists the ACP agents as programs with the models each reports.
func (s *agentServer) GetPrograms(ctx context.Context, requested []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	return s.lister.Execute(ctx, requested)
}
