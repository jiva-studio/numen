// Package wire translates between internal core types and protocol wire types.
package wire

import (
	"context"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// StreamAgentTask hands one task to the agent and writes what it does to the stream, for as
// long as the client listens.
//
// The work is stopped on the way out, whether it finished, failed or the client
// went away. Nothing outlives the panel it was asked from.
//
// A model thinking for a minute writes nothing, and a stream that writes
// nothing never learns its client has gone. A step naming nothing is one a
// client ignores and a write that fails ends the work.
func StreamAgentTask(
	ctx context.Context,
	taking agentport.Agent,
	task agentport.Task,
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	work, err := taking.StartRun(ctx, task)
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
		case step, working := <-work.GetSteps():
			if !working {
				return nil
			}
			if isDone, err := sendStep(stream, step); err != nil || isDone {
				return err
			}
		}
	}
}

// sendStep writes a step to the stream in the schema's words, and says whether
// the step was the last of the work.
func sendStep(stream *connect.ServerStream[v1.AskAgentResponse], step agentport.Step) (bool, error) {
	for _, out := range convertSteps(step) {
		if err := stream.Send(out); err != nil {
			return false, err
		}
	}
	return step.Kind == domain.StepStop, nil
}

// convertSteps says a step of an agent's work in the schema's words.
//
// Every kind that names a tool is drawn as one, whatever that tool does to the
// vault, and carries where in the vault it is working.
func convertSteps(step agentport.Step) []*v1.AskAgentResponse {
	switch step.Kind {
	case domain.StepToolCall, domain.StepRead, domain.StepEdit,
		domain.StepRemove, domain.StepMove, domain.StepSearch:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_ToolCall{
			ToolCall: &v1.ToolCall{
				Tool:   step.Tool,
				About:  step.About,
				Length: int32(step.Count),
				Path:   step.Place.Path,
				Span:   convertSpan(step.Place.Spans),
			},
		}}}
	case domain.StepToolDone:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_ToolFinished{ToolFinished: &v1.ToolFinished{}}}}
	case domain.StepModelStart:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_ModelStarted{ModelStarted: &v1.ModelStarted{}}}}
	case domain.StepStop:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_Stopped{Stopped: step.Detail}}}
	default:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_Text{Text: step.Text}}}
	}
}

// convertSpan is the one span a call names, and nothing where it named the source
// and no place inside it.
func convertSpan(spans []domain.ByteSpan) *v1.Span {
	if len(spans) == 0 {
		return nil
	}
	return &v1.Span{From: int32(spans[0].From), To: int32(spans[0].To)}
}
