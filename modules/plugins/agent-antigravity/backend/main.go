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
}

func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AskAgentRequest],
	stream *connect.ServerStream[numenv1.AskAgentResponse],
) error {
	if err := stream.Send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_ModelStarted{
			ModelStarted: &numenv1.ModelStarted{},
		},
	}); err != nil {
		return err
	}

	work, err := agentrun.Start(ctx, &s.Registry, newSpec(req.Msg, &s.Registry))
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
	if !IsAntigravityInstalled() {
		return connect.NewResponse(&numenv1.GetAgentStateResponse{
			Unreachable: "Antigravity was not found on this machine",
		}), nil
	}
	return connect.NewResponse(&numenv1.GetAgentStateResponse{}), nil
}

// GetAgentPrograms lists Antigravity and whether it is installed. It lists no models.
func (s *agentServer) GetAgentPrograms(
	_ context.Context,
	req *connect.Request[numenv1.GetAgentProgramsRequest],
) (*connect.Response[numenv1.GetAgentProgramsResponse], error) {
	isSet := false
	for _, one := range req.Msg.GetCommands() {
		if one.GetProgram() == "antigravity" && len(one.GetCommand()) > 0 {
			isSet = true
		}
	}
	antigravity := &numenv1.AgentProgram{Name: "antigravity", Title: "Antigravity", IsInstalled: IsAntigravityInstalled() || isSet}
	return connect.NewResponse(&numenv1.GetAgentProgramsResponse{
		Programs: []*numenv1.AgentProgram{antigravity},
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
