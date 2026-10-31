package service

import (
	"context"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/discovery"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/jsonlrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
)

type agentServer struct {
	agentrun.Service
	agentrun.ExecutablePath
	codexModels codexModelCache
}

func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[agentrun.AskAgentRequest],
	stream *connect.ServerStream[agentrun.AskAgentResponse],
) error {
	return agentrun.StreamWork(ctx, stream, func() (*agentrun.Work, error) {
		return jsonlrun.Start(ctx, &s.Registry, newSpec(req.Msg, s.GetExecutablePath()))
	})
}

func (s *agentServer) GetAgentState(
	_ context.Context,
	_ *connect.Request[agentrun.GetAgentStateRequest],
) (*connect.Response[agentrun.GetAgentStateResponse], error) {
	return agentrun.NewInstalledState(
		IsCodexInstalled(s.GetExecutablePath()), "OpenAI Codex was not found on this machine",
	), nil
}

// GetAgentPrograms lists OpenAI Codex, whether it is installed, and the models it lists.
func (s *agentServer) GetAgentPrograms(
	ctx context.Context,
	req *connect.Request[agentrun.GetAgentProgramsRequest],
) (*connect.Response[agentrun.GetAgentProgramsResponse], error) {
	command := discovery.GetCommand(req.Msg.GetCommands(), programName)
	customPath := s.GetExecutablePath()
	if len(command) == 0 && customPath != "" {
		command = []string{customPath}
	}

	codex := &agentrun.AgentProgram{
		Name: programName, Title: "OpenAI Codex", IsInstalled: IsCodexInstalled(customPath) || len(command) > 0,
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
