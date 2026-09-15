package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// errNoAgent is an installation with none. Whether one can be reached is asked
// of GetAgentState.
var errNoAgent = errors.New("no agent is set up for this vault")

// AskAgent hands the person's task to the agent and reports what it does for as
// long as the client listens.
func (a *API) AskAgent(
	ctx context.Context,
	r *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	taking := a.GetAgent()
	if taking == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errNoAgent)
	}

	return wire.Ask(ctx, taking, port.Task{
		Question:     r.Msg.GetAsked(),
		Focus:        r.Msg.GetFocus(),
		Conversation: r.Msg.GetConversation(),
	}, stream)
}

// FinishConversation says a conversation is over, and hands that on to the
// agent.
func (a *API) FinishConversation(
	ctx context.Context, r *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	taking := a.GetAgent()
	if taking == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoAgent)
	}

	if err := taking.Finish(ctx, r.Msg.GetConversation()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}

// GetAgentState is whether an agent can be reached at all. Which agent answers
// is the installation's, so every window asks it here.
func (a *API) GetAgentState(
	_ context.Context, _ *connect.Request[v1.GetAgentStateRequest],
) (*connect.Response[v1.GetAgentStateResponse], error) {
	return connect.NewResponse(&v1.GetAgentStateResponse{Unreachable: a.Unreachable.Why()}), nil
}
