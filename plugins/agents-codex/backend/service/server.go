package service

import (
	"context"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
)

type agentServer struct {
	agentrun.Service
	mu             sync.RWMutex
	executablePath string
	codexModels    codexModelCache
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

	work, err := start(ctx, &s.Registry, newSpec(req, s.getExecutablePath()))
	if err != nil {
		return send(&agentrun.Response{
			Step: &agentrun.StoppedStep{Stopped: err.Error()},
		})
	}
	return agentrun.ForwardResponses(ctx, work, send)
}

func (s *agentServer) GetState(_ context.Context) (*agentrun.State, error) {
	if !IsCodexInstalled(s.getExecutablePath()) {
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
		Name: "codex", Title: "OpenAI Codex", IsInstalled: IsCodexInstalled(customPath) || len(command) > 0,
		IsDefault: false, Models: nil, LimitsTools: false,
	}
	if codex.IsInstalled {
		for _, one := range s.codexModels.get(ctx, command) {
			codex.Models = append(codex.Models, &reasoning.Model{
				Name: one.Name, Title: one.Title, IsDefault: one.IsDefault,
				ReasoningLevels: one.Reasoning, DefaultReasoning: one.DefaultReasoning,
			})
		}
	}
	return []*agentrun.Program{codex}, nil
}
