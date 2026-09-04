package wire

import (
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// StepsOf says a step of an agent's work in the schema's words.
//
// Every kind that names a tool is drawn as one, whatever that tool does to the
// vault, and carries where in the vault it is working.
func StepsOf(step port.Step) []*v1.AskAgentResponse {
	switch step.Kind {
	case port.StepToolCall, port.StepRead, port.StepEdit,
		port.StepRemove, port.StepMove, port.StepSearch:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_ToolCall{
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
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_Answered{Answered: &v1.Answered{}}}}
	case port.StepThinking:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_Thinking{Thinking: &v1.Thinking{}}}}
	case port.StepStopped:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_Stopped{Stopped: step.Detail}}}
	default:
		return []*v1.AskAgentResponse{{Step: &v1.AskAgentResponse_Said{Said: step.Text}}}
	}
}
