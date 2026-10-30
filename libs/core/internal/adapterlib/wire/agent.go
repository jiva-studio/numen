// Package wire translates between internal core types and protocol wire types.
package wire

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	agentport "github.com/jiva-studio/numen/libs/core/usecase/completion/agentcontract"
)

// StreamAgentTask passes one task to the agent and writes its output to the
// stream for as long as the client is connected.
//
// The work is stopped on return, whether it finished, failed or the client
// disconnected. No work outlives the panel that requested it.
//
// A model that thinks for a minute writes nothing, and a stream that writes
// nothing never detects a disconnected client. A step with no content is ignored
// by the client, and a failed write ends the work.
func StreamAgentTask(
	ctx context.Context,
	taking agentport.Agent,
	task agentport.Task,
	stream *connect.ServerStream[v1.AskAgentResponse],
) (err error) {
	work, err := taking.StartRun(ctx, task)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	defer func() { err = errors.Join(err, work.Stop()) }()

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
				return finishStream(stream, "done")
			}
			if isDone, err := sendStep(stream, step); err != nil || isDone {
				return err
			}
		}
	}
}

// sendStep writes a step to the stream in the schema's representation and
// reports whether it was the last step of the work.
func sendStep(stream *connect.ServerStream[v1.AskAgentResponse], step agentport.Step) (bool, error) {
	if step.Kind == domain.StepStop {
		return true, finishStream(stream, step.Detail)
	}
	for _, out := range convertSteps(step) {
		if err := stream.Send(out); err != nil {
			return false, err
		}
	}
	return false, nil
}

// finishStream writes why the work stopped, followed by an empty frame that hands the stop to the
// window. An empty detail is work that finished.
func finishStream(stream *connect.ServerStream[v1.AskAgentResponse], detail string) error {
	if detail == "" {
		detail = "done"
	}
	if err := stream.Send(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_Stopped{Stopped: detail}}); err != nil {
		return err
	}
	return stream.Send(&v1.AskAgentResponse{})
}

// convertSteps converts a step of an agent's work to the schema's
// representation.
//
// Every kind that names a tool is sent as a tool step, whatever that tool does
// to the vault, and carries the vault location it works on.
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
				Range:  convertRange(step.Place.Ranges),
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

// convertRange is the one span a call names, and nothing where it named the source
// and no place inside it.
func convertRange(spans []domain.ByteRange) *v1.Range {
	if len(spans) == 0 {
		return nil
	}
	return &v1.Range{Start: int32(spans[0].Start), End: int32(spans[0].End)}
}
