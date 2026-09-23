package wire

import (
	"context"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Ask hands one task to the agent and writes what it does to the stream, for as
// long as the client listens.
//
// The work is stopped on the way out, whether it finished, failed or the client
// went away. Nothing outlives the panel it was asked from.
//
// A model thinking for a minute writes nothing, and a stream that writes
// nothing never learns its client has gone. A step naming nothing is one a
// client ignores and a write that fails ends the work.
func Ask(
	ctx context.Context,
	taking port.Agent,
	task port.Task,
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	work, err := taking.Take(ctx, task)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	defer work.Stop()

	repeat := time.NewTicker(Again)
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

// stepsOf says a step of an agent's work in the schema's words.
//
// Every kind that names a tool is drawn as one, whatever that tool does to the
// vault, and carries where in the vault it is working.
func stepsOf(step port.Step) []*v1.AskAgentResponse {
	switch step.Kind {
	case port.StepToolCall, port.StepRead, port.StepEdit,
		port.StepRemove, port.StepMove, port.StepSearch:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_ToolCall{
			ToolCall: &v1.ToolCall{
				Tool:    step.Tool,
				About:   step.About,
				Written: int32(step.Count),
				Path:    step.Place.Path,
				Span:    spanOf(step.Place.Spans),
			},
		}}}
	case port.StepAnswered:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_Answered{Answered: &v1.Answered{}}}}
	case port.StepThinking:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_Thinking{Thinking: &v1.Thinking{}}}}
	case port.StepStopped:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_Stopped{Stopped: step.Detail}}}
	default:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_Said{Said: step.Text}}}
	}
}

// spanOf is the one span a call names, and nothing where it named the source
// and no place inside it.
func spanOf(spans []domain.ByteSpan) *v1.Span {
	if len(spans) == 0 {
		return nil
	}
	return &v1.Span{From: int32(spans[0].From), To: int32(spans[0].To)}
}
