package webui

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// AskAgent hands the person's task to the agent and reports what it does for as
// long as the client listens.
//
// The work is stopped on the way out, whether it finished, failed or the client
// went away. Nothing outlives the panel it was asked from.
func (a *API) AskAgent(
	ctx context.Context,
	r *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	taking := a.Answering()
	if taking == nil {
		return connect.NewError(connect.CodeUnimplemented,
			errors.New("no agent is set up for this vault"))
	}

	work, err := taking.Take(ctx, port.Task{
		Question:     r.Msg.GetAsked(),
		Focus:        r.Msg.GetFocus(),
		Conversation: r.Msg.GetConversation(),
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	defer work.Stop()

	// A model thinking for a minute writes nothing, and a stream that writes
	// nothing never learns its client has gone. A step naming nothing is one a
	// client ignores and a write that fails ends the work.
	repeat := time.NewTicker(wire.Again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-repeat.C:
			if err := stream.Send(&v1.AskAgentResponse{}); err != nil {
				return err
			}
		case step, working := <-work.Steps():
			if !working {
				return nil
			}
			for _, out := range wire.StepsOf(step) {
				if err := stream.Send(out); err != nil {
					return err
				}
			}
			if step.Kind == port.StepStopped {
				return nil
			}
		}
	}
}

// FinishConversation says a conversation is over, and hands that on to the
// agent.
func (a *API) FinishConversation(
	ctx context.Context, r *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	taking := a.Answering()
	if taking == nil {
		return nil, connect.NewError(connect.CodeUnimplemented,
			errors.New("no agent is set up for this vault"))
	}

	if err := taking.Finish(ctx, r.Msg.GetConversation()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}
