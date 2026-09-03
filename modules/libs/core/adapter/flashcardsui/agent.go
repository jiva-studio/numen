package flashcardsui

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrNoAgent is a question asked at a window that can reach none.
var ErrNoAgent = errors.New("no agent is set up for this window")

// Asking is whether a card can be asked about here, and on which cards the way
// in is offered.
//
// It answers what this window can reach and not what it has reached: the page
// asks as it opens, and the agent is started when a person sits down to a
// vault.
func (a *API) Asking(
	_ context.Context, _ *connect.Request[v1.AskingRequest],
) (*connect.Response[v1.AskingResponse], error) {
	why, _ := a.Unreachable.Load().(string)
	return connect.NewResponse(&v1.AskingResponse{Unreachable: why}), nil
}

// Ask hands the person's question to the agent and reports what it does for as
// long as the client listens.
//
// The work is stopped on the way out, whether it finished, failed or the client
// went away. Nothing outlives the card it was asked about.
func (a *API) Ask(
	ctx context.Context,
	r *connect.Request[v1.AskRequest],
	stream *connect.ServerStream[v1.AskResponse],
) error {
	taking := a.Answering()
	if taking == nil {
		return connect.NewError(connect.CodeUnimplemented, ErrNoAgent)
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

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case step, working := <-work.Steps():
			if !working {
				return nil
			}
			for _, out := range stepsOf(step) {
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

// Finish says a conversation is over, and hands that on to the agent.
func (a *API) Finish(
	ctx context.Context, r *connect.Request[v1.FinishRequest],
) (*connect.Response[v1.FinishResponse], error) {
	taking := a.Answering()
	if taking == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, ErrNoAgent)
	}

	if err := taking.Finish(ctx, r.Msg.GetConversation()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.FinishResponse{}), nil
}

// stepsOf says a step in the schema's words.
//
// Every kind that names a tool is drawn as one, and carries where in the vault
// it is working.
func stepsOf(step port.Step) []*v1.AskResponse {
	switch step.Kind {
	case port.StepToolCall, port.StepRead, port.StepEdit,
		port.StepRemove, port.StepMove, port.StepSearch:
		return []*v1.AskResponse{{Step: &v1.AskResponse_ToolCall{
			ToolCall: &v1.ToolCall{
				Tool:    step.Tool,
				About:   step.About,
				Written: int32(step.Count),
				Path:    step.Place.Path,
				Start:   int32(step.Place.Start),
				Length:  int32(step.Place.Length),
			},
		}}}
	case port.StepAnswered:
		return []*v1.AskResponse{{Step: &v1.AskResponse_Answered{Answered: &v1.Answered{}}}}
	case port.StepThinking:
		return []*v1.AskResponse{{Step: &v1.AskResponse_Thinking{Thinking: &v1.Thinking{}}}}
	case port.StepStopped:
		return []*v1.AskResponse{{Step: &v1.AskResponse_Stopped{Stopped: step.Detail}}}
	default:
		return []*v1.AskResponse{{Step: &v1.AskResponse_Said{Said: step.Text}}}
	}
}
