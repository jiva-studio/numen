package service

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/jsonfield"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/jsonlrun"
)

// newEventHandler is what the run calls with each event of the Codex output.
func newEventHandler(r *jsonlrun.Reader) func(map[string]any) {
	return func(event map[string]any) { handleCodexEvent(r, event) }
}

func handleCodexEvent(r *jsonlrun.Reader, rawMap map[string]any) {
	evType := jsonfield.GetString(rawMap, "type")
	item := jsonfield.GetMap(rawMap, "item")
	if item == nil {
		return
	}

	switch {
	case evType == "item.completed" && item["type"] == "agent_message":
		if text := jsonfield.GetString(item, "text"); text != "" {
			r.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseText{Text: text}})
		}
	case evType == "item.started" && item["type"] == "mcp_tool_call":
		r.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponseToolCall{
				ToolCall: &agentrun.ToolCall{
					Tool:   getToolName(item),
					About:  getToolArguments(item),
					Length: 0,
					Path:   "",
					Range:  nil,
					Call:   "",
					Kind:   "",
				},
			},
		})
	case evType == "item.completed" && item["type"] == "mcp_tool_call":
		r.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponseToolFinished{
				ToolFinished: &agentrun.ToolFinished{},
			},
		})
	}
}

// getToolName is the tool an mcp_tool_call item names: its tool, else its name.
func getToolName(item map[string]any) string {
	if tool := jsonfield.GetString(item, "tool"); tool != "" {
		return tool
	}
	name := jsonfield.GetString(item, "name")
	return name
}

func getToolArguments(item map[string]any) string {
	args, err := json.Marshal(item["arguments"])
	if err != nil {
		return ""
	}
	return string(args)
}
