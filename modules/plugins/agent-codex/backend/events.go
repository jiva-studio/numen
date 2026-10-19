package main

import (
	"encoding/json"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// newEventHandler is what the run calls with each event of the Codex output.
func newEventHandler(r *agentrun.Reader) func(map[string]any) {
	return func(event map[string]any) { handleCodexEvent(r, event) }
}

func handleCodexEvent(r *agentrun.Reader, rawMap map[string]any) {
	evType, _ := rawMap["type"].(string)
	item, _ := rawMap["item"].(map[string]any)
	if item == nil || item["type"] != "mcp_tool_call" {
		return
	}

	switch evType {
	case "item.started":
		name, _ := item["name"].(string)
		args, _ := json.Marshal(item["arguments"])
		r.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_ToolCall{
				ToolCall: &numenv1.ToolCall{
					Tool:  name,
					About: string(args),
				},
			},
		})
	case "item.completed":
		r.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_ToolFinished{
				ToolFinished: &numenv1.ToolFinished{},
			},
		})
	}
}
