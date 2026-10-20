package service

import (
	"context"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

type agentServer struct {
	agentrun.Service
	models agentrun.ModelCache[antigravityModel]
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
	return agentrun.ForwardResponses(ctx, work, stream)
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

// GetAgentPrograms lists Antigravity, whether it is installed, and the models it lists.
func (s *agentServer) GetAgentPrograms(
	ctx context.Context,
	req *connect.Request[numenv1.GetAgentProgramsRequest],
) (*connect.Response[numenv1.GetAgentProgramsResponse], error) {
	var command []string
	for _, one := range req.Msg.GetCommands() {
		if one.GetProgram() == "antigravity" {
			command = one.GetCommand()
		}
	}
	antigravity := &numenv1.AgentProgram{
		Name: "antigravity", Title: "Antigravity", IsInstalled: IsAntigravityInstalled() || len(command) > 0,
	}
	if antigravity.GetIsInstalled() {
		list := func(ctx context.Context) ([]antigravityModel, error) { return listAntigravityModels(ctx, command) }
		for _, one := range s.models.Get(ctx, strings.Join(command, " "), list) {
			antigravity.Models = append(antigravity.Models, &numenv1.AgentModel{Name: one.Name, Title: one.Title})
		}
	}
	return connect.NewResponse(&numenv1.GetAgentProgramsResponse{
		Programs: []*numenv1.AgentProgram{antigravity},
	}), nil
}
