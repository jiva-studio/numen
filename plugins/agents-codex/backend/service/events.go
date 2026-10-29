package service

import (
	"encoding/json"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// newEventHandler is what the run calls with each event of the Codex output.
func newEventHandler(r *reader) func(map[string]any) {
	return func(event map[string]any) { handleCodexEvent(r, event) }
}

func handleCodexEvent(r *reader, rawMap map[string]any) {
	evType, _ := rawMap["type"].(string)
	item, _ := rawMap["item"].(map[string]any)
	if item == nil {
		return
	}

	switch {
	case evType == "item.completed" && item["type"] == "agent_message":
		if text, _ := item["text"].(string); text != "" {
			r.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponse_Text{Text: text}})
		}
	case evType == "item.started" && item["type"] == "mcp_tool_call":
		r.Send(&agentrun.AskAgentResponse{
			Step: &agentrun.AskAgentResponse_ToolCall{
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
			Step: &agentrun.AskAgentResponse_ToolFinished{
				ToolFinished: &agentrun.ToolFinished{},
			},
		})
	}
}

// getToolName is the tool an mcp_tool_call item names: its tool, else its name.
func getToolName(item map[string]any) string {
	if tool, _ := item["tool"].(string); tool != "" {
		return tool
	}
	name, _ := item["name"].(string)
	return name
}

func getToolArguments(item map[string]any) string {
	args, _ := json.Marshal(item["arguments"])
	return string(args)
}
