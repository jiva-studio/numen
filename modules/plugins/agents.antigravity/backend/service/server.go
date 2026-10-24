package service

import (
	"context"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/reasoning"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

type agentServer struct {
	agentrun.Service
	models modelCache[antigravityModel]
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

	s.resolveModel(ctx, req.Msg)
	work, err := start(ctx, &s.Registry, newSpec(req.Msg, &s.Registry))
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
	command := programCommand(req.Msg.GetCommands(), "antigravity")
	antigravity := &numenv1.AgentProgram{
		Name: "antigravity", Title: "Antigravity", IsInstalled: IsAntigravityInstalled() || len(command) > 0,
	}
	if antigravity.GetIsInstalled() {
		for _, one := range s.getFamilies(ctx, command) {
			antigravity.Models = append(antigravity.Models, one.ToProto())
		}
	}
	return connect.NewResponse(&numenv1.GetAgentProgramsResponse{
		Programs: []*numenv1.AgentProgram{antigravity},
	}), nil
}

// getFamilies are the models the program lists, the ones that differ by level folded into one.
func (s *agentServer) getFamilies(ctx context.Context, command []string) []reasoning.Family {
	list := func(ctx context.Context) ([]antigravityModel, error) { return listAntigravityModels(ctx, command) }
	items := s.models.get(ctx, strings.Join(command, " "), list)
	models := make([]reasoning.Listed, 0, len(items))
	for _, one := range items {
		models = append(models, reasoning.Listed{Name: one.Name, Title: one.Title})
	}
	return reasoning.Fold(models)
}

// resolveModel sets the model of the request to the one the program lists for the model and the level
// the request names.
func (s *agentServer) resolveModel(ctx context.Context, req *numenv1.AskAgentRequest) {
	if req.GetModel() == "" || req.GetModel() == "default" {
		return
	}
	req.Model = reasoning.ResolveModel(s.getFamilies(ctx, req.GetCommand()), req.GetModel(), req.GetReasoning())
}
