package main

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type agentServer struct {
	agentrun.Service
	codexModels codexModelCache
}

func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AskAgentRequest],
	stream *connect.ServerStream[numenv1.AskAgentResponse],
) error {
	_ = stream.Send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_ModelStarted{
			ModelStarted: &numenv1.ModelStarted{},
		},
	})

	work, err := agentrun.Start(ctx, &s.Registry, newSpec(req.Msg))
	if err != nil {
		return stream.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Stopped{Stopped: err.Error()},
		})
	}
	return agentrun.Relay(ctx, work, stream)
}

func (s *agentServer) GetAgentState(
	_ context.Context,
	_ *connect.Request[numenv1.GetAgentStateRequest],
) (*connect.Response[numenv1.GetAgentStateResponse], error) {
	if !IsCodexInstalled() {
		return connect.NewResponse(&numenv1.GetAgentStateResponse{
			Unreachable: "OpenAI Codex was not found on this machine",
		}), nil
	}
	return connect.NewResponse(&numenv1.GetAgentStateResponse{}), nil
}

// GetAgentPrograms lists OpenAI Codex, whether it is installed, and the models it lists.
func (s *agentServer) GetAgentPrograms(
	ctx context.Context,
	req *connect.Request[numenv1.GetAgentProgramsRequest],
) (*connect.Response[numenv1.GetAgentProgramsResponse], error) {
	var command []string
	for _, one := range req.Msg.GetCommands() {
		if one.GetProgram() == "codex" {
			command = one.GetCommand()
		}
	}

	codex := &numenv1.AgentProgram{Name: "codex", Title: "OpenAI Codex", IsInstalled: IsCodexInstalled() || len(command) > 0}
	if codex.GetIsInstalled() {
		for _, one := range s.codexModels.get(ctx, command) {
			codex.Models = append(codex.Models, &numenv1.AgentModel{
				Name: one.Name, Title: one.Title, IsDefault: one.IsDefault,
			})
		}
	}
	return connect.NewResponse(&numenv1.GetAgentProgramsResponse{
		Programs: []*numenv1.AgentProgram{codex},
	}), nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	s := server.New()

	agentSvc := &agentServer{}
	defer agentSvc.Close()

	s.Register(numenv1connect.NewAgentServiceHandler(agentSvc, s.HandlerOptions()...))

	return s.Serve()
}
