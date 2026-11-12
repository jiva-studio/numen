package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/codexcli"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/modellist"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/programs"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/run"
)

type agentServer struct {
	agentrun.Service
	settings *settings.Settings
	programs *programs.Programs
	runner   *run.Runner
}

func newAgentServer(isAsync bool) *agentServer {
	models := modellist.NewCache(codexcli.Models{}, clock.System{}, isAsync)
	return &agentServer{
		Service:  agentrun.Service{},
		settings: new(settings.Settings),
		programs: programs.New(codexcli.Installation{}, models),
		runner:   run.NewRunner(codexcli.Commands{}, codexcli.Launcher{}),
	}
}

func (s *agentServer) Ask(ctx context.Context, req *agentrun.Request, send func(*agentrun.Response) error) error {
	return s.runner.Ask(ctx, &s.Registry, req, s.settings.ExecutablePath(), send)
}

func (s *agentServer) GetState(_ context.Context) (*agentrun.State, error) {
	return s.programs.GetState(s.settings.ExecutablePath()), nil
}

// GetPrograms lists OpenAI Codex, whether it is installed, and the models it lists.
func (s *agentServer) GetPrograms(ctx context.Context, commands []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	return s.programs.GetPrograms(ctx, s.settings.ExecutablePath(), commands), nil
}
