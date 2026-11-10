package service

import (
	"context"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/codexcli"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/modellist"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/run"
)

type agentServer struct {
	agentrun.Service
	mu             sync.RWMutex
	executablePath string
	models         *modellist.Cache
	runner         *run.Runner
}

func newAgentServer(isAsync bool) *agentServer {
	return &agentServer{
		Service:        agentrun.Service{},
		mu:             sync.RWMutex{},
		executablePath: "",
		models:         modellist.NewCache(codexcli.Models{}, clock.System{}, isAsync),
		runner:         run.NewRunner(codexcli.Commands{}, process.Group{}),
	}
}

func (s *agentServer) applyConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execPath, ok := values["executablePath"]; ok {
		s.executablePath = strings.TrimSpace(execPath)
	}
}

func (s *agentServer) getExecutablePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.executablePath
}

func (s *agentServer) Ask(ctx context.Context, req *agentrun.Request, send func(*agentrun.Response) error) error {
	if err := send(&agentrun.Response{Step: &agentrun.ModelStartedStep{}}); err != nil {
		return err
	}

	work, err := s.runner.Start(ctx, &s.Registry, req, s.getExecutablePath())
	if err != nil {
		return send(&agentrun.Response{
			Step: &agentrun.StoppedStep{Stopped: err.Error()},
		})
	}
	return agentrun.ForwardResponses(ctx, work, send)
}

func (s *agentServer) GetState(_ context.Context) (*agentrun.State, error) {
	if !codexcli.IsCodexInstalled(s.getExecutablePath()) {
		return &agentrun.State{Unreachable: "OpenAI Codex was not found on this machine"}, nil
	}
	return &agentrun.State{Unreachable: ""}, nil
}

// GetPrograms lists OpenAI Codex, whether it is installed, and the models it lists.
func (s *agentServer) GetPrograms(ctx context.Context, commands []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	command := programCommand(commands, "codex")
	customPath := s.getExecutablePath()
	if len(command) == 0 && customPath != "" {
		command = []string{customPath}
	}

	codex := &agentrun.Program{
		Name: "codex", Title: "OpenAI Codex", IsInstalled: codexcli.IsCodexInstalled(customPath) || len(command) > 0,
		IsDefault: false, Models: nil, LimitsTools: false,
	}
	if codex.IsInstalled {
		for _, one := range s.models.Get(ctx, command) {
			codex.Models = append(codex.Models, &reasoning.Model{
				Name: one.Name, Title: one.Title, IsDefault: one.IsDefault,
				ReasoningLevels: one.Reasoning, DefaultReasoning: one.DefaultReasoning,
			})
		}
	}
	return []*agentrun.Program{codex}, nil
}

func programCommand(commands []agentrun.ProgramCommand, name string) []string {
	for _, one := range commands {
		if one.Program == name {
			return one.Command
		}
	}
	return nil
}
