package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/catalog"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/internal/adapter/antigravity"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/runs"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/settings"
)

type agentServer struct {
	agentrun.Service
	settings *settings.Settings
	catalog  *catalog.Catalog
	runner   *runs.Runner
}

func newAgentServer(isAsync bool) *agentServer {
	supervisor := process.Supervisor{}
	cli := antigravity.New(supervisor)
	s := new(agentServer)
	s.settings = new(settings.Settings)
	s.catalog = catalog.New(cli, cli, isAsync)
	s.runner = runs.New(&s.Registry, cli, process.Launcher{}, s.catalog)
	return s
}

func (s *agentServer) Ask(ctx context.Context, req *agentrun.Request, send func(*agentrun.Response) error) error {
	return s.runner.Ask(ctx, req, s.settings.ExecutablePath(), send)
}

func (s *agentServer) GetState(_ context.Context) (*agentrun.State, error) {
	return s.catalog.GetState(s.settings.ExecutablePath()), nil
}

func (s *agentServer) GetPrograms(ctx context.Context, commands []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	return s.catalog.GetPrograms(ctx, s.settings.ExecutablePath(), commands), nil
}
