package agentrun

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// NewHandler returns the path and the HTTP handler that serve the agent over the agent service.
func NewHandler(agent Agent, opts ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewAgentServiceHandler(&handler{agent: agent}, opts...)
}

// handler answers the calls of the agent service with an Agent.
type handler struct {
	agent Agent
}

func (h *handler) AskAgent(
	ctx context.Context,
	req *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	return h.agent.Ask(ctx, requestFromProto(req.Msg), func(resp *Response) error {
		return stream.Send(responseToProto(resp))
	})
}

func (h *handler) FinishConversation(
	ctx context.Context,
	req *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	if err := h.agent.FinishConversation(ctx, req.Msg.GetConversation()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}

func (h *handler) GetAgentState(
	ctx context.Context,
	_ *connect.Request[v1.GetAgentStateRequest],
) (*connect.Response[v1.GetAgentStateResponse], error) {
	state, err := h.agent.GetState(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(stateToProto(state)), nil
}

func (h *handler) GetAgentPrograms(
	ctx context.Context,
	req *connect.Request[v1.GetAgentProgramsRequest],
) (*connect.Response[v1.GetAgentProgramsResponse], error) {
	programs, err := h.agent.GetPrograms(ctx, programCommandsFromProto(req.Msg))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(programsToProto(programs)), nil
}

func (h *handler) RegisterTool(
	ctx context.Context,
	req *connect.Request[v1.RegisterToolRequest],
) (*connect.Response[v1.RegisterToolResponse], error) {
	if err := h.agent.RegisterTool(ctx, toolDefinitionFromProto(req.Msg.GetTool())); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.RegisterToolResponse{Success: true}), nil
}

func (h *handler) ExtendSystemPrompt(
	ctx context.Context,
	req *connect.Request[v1.ExtendSystemPromptRequest],
) (*connect.Response[v1.ExtendSystemPromptResponse], error) {
	full, err := h.agent.ExtendSystemPrompt(ctx, req.Msg.GetExtension())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ExtendSystemPromptResponse{Success: true, FullPrompt: full}), nil
}

func (h *handler) GetAgentTools(
	ctx context.Context,
	_ *connect.Request[v1.GetAgentToolsRequest],
) (*connect.Response[v1.GetAgentToolsResponse], error) {
	set, err := h.agent.GetTools(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(toolSetToProto(set)), nil
}
