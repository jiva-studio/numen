package answer

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
)

// describeCall is the tool call frame of a call: the title of the listed tool, its kind by the effect of
// the tool, what the call is about and the note it works on, read from the arguments.
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
	about, path := readSubject(call.Arguments)
	frame.Content = []string{about}
	if path != "" {
		frame.Locations = []*acp.ToolCallLocation{{Path: path, Line: 0}}
	}
	return frame
}

// readSubject reads what a call is about (its query, about or title, else its path) and the path of the
// note it names, from the arguments the tools of the host use for them.
func readSubject(arguments string) (about, path string) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(arguments), &parsed); err != nil {
		return "", ""
	}
	path = readFirstString(parsed, "path", "file")
	about = readFirstString(parsed, "query", "about", "title")
	if about == "" {
		about = path
	}
	return about, path
}

func readFirstString(parsed map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, isText := parsed[key].(string); isText && value != "" {
			return value
		}
	}
	return ""
}
