package flashcards

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrNoAgent is a question asked at a window that can reach none. Whether one
// can be reached is a fact about this window at this moment, and GetAgentState
// answers it.
var ErrNoAgent = errors.New("no agent is set up for this window")

// GetAgentState is whether a card can be asked about here, and on which cards
// the way in is offered.
//
// It answers what this window can reach and not what it has reached: the page
// asks as it opens, and the agent is started when a person sits down to a
// vault.
func (a *API) GetAgentState(
	_ context.Context, _ *connect.Request[v1.GetAgentStateRequest],
) (*connect.Response[v1.GetAgentStateResponse], error) {
	return connect.NewResponse(&v1.GetAgentStateResponse{Unreachable: a.Unreachable.Why()}), nil
}

// AskAgent hands the person's question to the agent and reports what it does
// for as long as the client listens.
func (a *API) AskAgent(
	ctx context.Context,
	r *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	taking := a.Answering()
	if taking == nil {
		return connect.NewError(connect.CodeFailedPrecondition, ErrNoAgent)
	}

	// The card is what `card_showing` answers with, not part of the question. A
	// deck in a synced vault is named by whoever synced it, and a name written
	// into the question is read as instruction where a tool's answer is data.
	a.showing.Store(&CurrentCard{
		Deck: r.Msg.GetFocus(),
		Card: r.Msg.GetMark(),
		Face: r.Msg.GetFace(),
	})

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
	taking := a.Answering()
	if taking == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrNoAgent)
	}

	if err := taking.Finish(ctx, r.Msg.GetConversation()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}
