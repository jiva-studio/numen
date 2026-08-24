package webui

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agent"
)

// Ask hands the person's task to the agent and reports what it does for as long
// as the client listens.
//
// The work is stopped on the way out, whether it finished, failed or the client
// went away. Nothing outlives the panel it was asked from.
func (a *API) Ask(ctx context.Context, r *connect.Request[v1.AskRequest], stream *connect.ServerStream[v1.AskResponse]) error {
	taking := a.Answering()
	if taking == nil {
		return connect.NewError(connect.CodeUnimplemented,
			errors.New("no agent is set up for this vault"))
	}

	work, err := taking.Take(ctx, agent.Task{
		Asked:        r.Msg.GetAsked(),
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
			if step.Kind == agent.Stopped {
				return nil
			}
		}
	}
}

// Finish says a conversation is over, and hands that on to the agent.
func (a *API) Finish(ctx context.Context, r *connect.Request[v1.FinishRequest]) (*connect.Response[v1.FinishResponse], error) {
	taking := a.Answering()
	if taking == nil {
		return nil, connect.NewError(connect.CodeUnimplemented,
			errors.New("no agent is set up for this vault"))
	}

	if err := taking.Finish(ctx, r.Msg.GetConversation()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.FinishResponse{}), nil
}

// stepsOf says a step in the schema's words.
//
// Every kind that names a tool is drawn as one, whatever that tool does to the
// vault, and carries where in the vault it is working.
func stepsOf(step agent.Step) []*v1.AskResponse {
	switch step.Kind {
	case agent.Calling, agent.Read, agent.Edit, agent.Remove, agent.Move, agent.Search:
		return []*v1.AskResponse{{Step: &v1.AskResponse_Doing{
			Doing: &v1.Doing{
				Tool:    step.Tool,
				About:   step.About,
				Written: int32(step.Written),
				Path:    step.Place.Path,
				Start:   int32(step.Place.Start),
				Length:  int32(step.Place.Length),
			},
		}}}
	case agent.Answered:
		return []*v1.AskResponse{{Step: &v1.AskResponse_Answered{Answered: &v1.Answered{}}}}
	case agent.Thinking:
		return []*v1.AskResponse{{Step: &v1.AskResponse_Thinking{Thinking: &v1.Thinking{}}}}
	case agent.Stopped:
		return []*v1.AskResponse{{Step: &v1.AskResponse_Stopped{Stopped: step.Failed}}}
	default:
		return []*v1.AskResponse{{Step: &v1.AskResponse_Said{Said: step.Text}}}
	}
}
