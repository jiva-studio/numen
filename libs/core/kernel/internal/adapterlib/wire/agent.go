// Package wire translates between internal core types and protocol wire types.
package wire

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	agentport "github.com/jiva-studio/numen/libs/core/kernel/usecase/completion/agentcontract"
)

// StreamAgentTask passes one task to the agent and writes its output to the
// stream while the client is connected. The work stops on return, so none
// outlives the panel that requested it.
// A model that thinks for a minute writes nothing and a silent stream never
// detects a disconnect, so a content-free step is sent; a failed write ends the work.
func StreamAgentTask(
	ctx context.Context,
	taking agentport.Agent,
	task agent.Task,
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

func sendStep(stream *connect.ServerStream[v1.AskAgentResponse], step agent.Step) (bool, error) {
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

func finishStream(stream *connect.ServerStream[v1.AskAgentResponse], detail string) error {
	if detail == "" {
		detail = "done"
	}
	if err := stream.Send(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_Stopped{Stopped: detail}}); err != nil {
		return err
	}
	return stream.Send(&v1.AskAgentResponse{})
}

func convertSteps(step agent.Step) []*v1.AskAgentResponse {
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

func convertRange(spans []domain.ByteRange) *v1.Range {
	if len(spans) == 0 {
		return nil
	}
	return &v1.Range{Start: int32(spans[0].Start), End: int32(spans[0].End)}
}
