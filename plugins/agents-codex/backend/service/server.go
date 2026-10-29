package service

import (
	"context"
	"strings"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
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
	if execPath, ok := values["executable_path"]; ok {
		s.executablePath = strings.TrimSpace(execPath)
	}
}

func (s *agentServer) getExecutablePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.executablePath
}

func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[agentrun.AskAgentRequest],
	stream *connect.ServerStream[agentrun.AskAgentResponse],
) error {
	if err := stream.Send(&agentrun.AskAgentResponse{
		Step: &agentrun.AskAgentResponse_ModelStarted{
			ModelStarted: &agentrun.ModelStarted{},
		},
	}); err != nil {
		return err
	}

	work, err := start(ctx, &s.Registry, newSpec(req.Msg, s.getExecutablePath()))
	if err != nil {
		return stream.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponse_Stopped{Stopped: err.Error()},
		})
	}
	return agentrun.ForwardResponses(ctx, work, stream)
}

func (s *agentServer) GetAgentState(
	_ context.Context,
	_ *connect.Request[agentrun.GetAgentStateRequest],
) (*connect.Response[agentrun.GetAgentStateResponse], error) {
	if !IsCodexInstalled(s.getExecutablePath()) {
		return connect.NewResponse(&agentrun.GetAgentStateResponse{
			Unreachable: "OpenAI Codex was not found on this machine",
		}), nil
	}
	return connect.NewResponse(&agentrun.GetAgentStateResponse{Unreachable: ""}), nil
}

// GetAgentPrograms lists OpenAI Codex, whether it is installed, and the models it lists.
func (s *agentServer) GetAgentPrograms(
	ctx context.Context,
	req *connect.Request[agentrun.GetAgentProgramsRequest],
) (*connect.Response[agentrun.GetAgentProgramsResponse], error) {
	command := programCommand(req.Msg.GetCommands(), "codex")
	customPath := s.getExecutablePath()
	if len(command) == 0 && customPath != "" {
		command = []string{customPath}
	}

	codex := &agentrun.AgentProgram{
		Name: "codex", Title: "OpenAI Codex", IsInstalled: IsCodexInstalled(customPath) || len(command) > 0,
		IsDefault: false, Models: nil, LimitsTools: false,
	}
	if codex.GetIsInstalled() {
		for _, one := range s.codexModels.get(ctx, command) {
			codex.Models = append(codex.Models, &agentrun.AgentModel{
				Name: one.Name, Title: one.Title, IsDefault: one.IsDefault,
				ReasoningLevels: reasoning.ToProto(one.Reasoning), DefaultReasoning: one.DefaultReasoning,
			})
		}
	}
	return connect.NewResponse(&agentrun.GetAgentProgramsResponse{
		Programs: []*agentrun.AgentProgram{codex},
	}), nil
}
