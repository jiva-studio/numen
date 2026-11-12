package answer

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
)

func describeCall(listed map[string]conversation.Tool, call conversation.ToolCall) *acp.ToolCall {
	frame := &acp.ToolCall{
		ToolCallId:   call.ID,
		Title:        call.Name,
		Kind:         acp.ToolKind_TOOL_KIND_OTHER,
		Status:       acp.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS,
		RawInputJson: call.Arguments,
	}
	if tool, isListed := listed[call.Name]; isListed {
		if tool.Title != "" {
			frame.Title = tool.Title
		}
		frame.Kind = acp.ToolKind_TOOL_KIND_EDIT
		if tool.IsReadOnly {
			frame.Kind = acp.ToolKind_TOOL_KIND_READ
		}
	}
	frame.Content = []string{call.About}
	if call.Path != "" {
		frame.Locations = []*acp.ToolCallLocation{{Path: call.Path, Line: 0}}
	}
	return frame
}
