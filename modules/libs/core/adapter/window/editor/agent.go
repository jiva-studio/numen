package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/agent"
)

// agentHandler answers for the agent the panel's tasks go to.
type agentHandler struct {
	numenv1connect.UnimplementedAgentServiceHandler
	state *State
}

// errNoAgent is an installation with none. Whether one can be reached is asked
// of GetAgentState.
var errNoAgent = errors.New("no agent is set up for this vault")

// AskAgent hands the person's task to the agent and reports what it does for as
// long as the client listens.
func (h *agentHandler) AskAgent(
	ctx context.Context,
	r *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	taking := h.state.GetAgent()
	if taking == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errNoAgent)
	}

	task := port.Task{
		Question:     r.Msg.GetPrompt(),
		Focus:        r.Msg.GetFocus(),
		Conversation: r.Msg.GetConversation(),
		Agent:        r.Msg.GetAgent(),
		Model:        r.Msg.GetModel(),
		Reasoning:    r.Msg.GetReasoning(),
	}
	task.Instructions = agent.GetInstructions(task)
	return wire.StreamAgentTask(ctx, taking, task, stream)
}

// FinishConversation says a conversation is over, and hands that on to the
// agent.
func (h *agentHandler) FinishConversation(
	ctx context.Context, r *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	taking := h.state.GetAgent()
	if taking == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoAgent)
	}

	if err := taking.EndConversation(ctx, r.Msg.GetConversation()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}

// GetAgentState is whether an agent can be reached at all. Which agent answers
// is the installation's, so every window asks it here.
func (h *agentHandler) GetAgentState(
	_ context.Context, _ *connect.Request[v1.GetAgentStateRequest],
) (*connect.Response[v1.GetAgentStateResponse], error) {
	return connect.NewResponse(&v1.GetAgentStateResponse{Unreachable: h.state.Unreachable.Why()}), nil
}
